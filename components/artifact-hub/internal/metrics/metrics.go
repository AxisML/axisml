package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registry is artifact-hub's own collector registry. It is independent of any
// controller-runtime registry — the service runs no manager — and is served
// directly via Handler.
var registry = prometheus.NewRegistry()

var (
	// IsLeader is set to 1 while this replica runs the GC worker, i.e. holds
	// the GC advisory lock (or runs unconditionally when leader election is
	// disabled).
	IsLeader = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "axisml_artifacts_is_leader",
		Help: "1 when this replica currently runs the GC worker (holds the GC advisory lock).",
	})

	// HTTPRequestDuration captures per-route timing.
	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "axisml_artifacts_api_request_duration_seconds",
		Help:    "HTTP API request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "method", "status"})

	// UploadingCount is the count of artifacts in the Uploading state, by Kind.
	// Required by design §2.5.
	UploadingCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "axisml_artifacts_uploading_count",
		Help: "Number of artifacts currently in Uploading state, partitioned by kind.",
	}, []string{"kind"})

	// GCActions counts GC dispatch outcomes.
	GCActions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axisml_artifacts_gc_actions_total",
		Help: "GC action outcomes.",
	}, []string{"predicate", "result"})

	// ResolveRequests counts resolve requests, partitioned by kind and result.
	ResolveRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axisml_artifacts_resolve_requests_total",
		Help: "resolve request outcomes.",
	}, []string{"kind", "usage", "result"})
)

// Register installs all collectors into the service registry. Call once at
// startup before Handler is served.
func Register() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		IsLeader,
		HTTPRequestDuration,
		UploadingCount,
		GCActions,
		ResolveRequests,
	)
}

// Handler serves the registered collectors in Prometheus text format. Mount it
// on the metrics listener (GET /metrics).
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// GinMiddleware records request duration histograms.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		HTTPRequestDuration.
			WithLabelValues(route, c.Request.Method, strconv.Itoa(c.Writer.Status())).
			Observe(time.Since(start).Seconds())
	}
}
