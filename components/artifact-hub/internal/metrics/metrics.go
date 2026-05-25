package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// IsLeader is set to 1 once this replica wins leader election.
	IsLeader = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "axisml_artifacts_is_leader",
		Help: "1 when this replica currently holds the controller-runtime leader lease.",
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

// Register installs all collectors into the controller-runtime metrics
// registry (which the manager also serves on /metrics).
func Register() {
	ctrlmetrics.Registry.MustRegister(
		IsLeader,
		HTTPRequestDuration,
		UploadingCount,
		GCActions,
		ResolveRequests,
	)
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
