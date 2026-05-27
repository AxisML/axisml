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
		Name: "axisml_compute_is_leader",
		Help: "1 when this replica currently holds the controller-runtime leader lease.",
	})

	// HTTPRequestDuration captures per-route timing.
	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "axisml_compute_api_request_duration_seconds",
		Help:    "HTTP API request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "method", "status"})

	// ReconcilerActions counts reconciler dispatch outcomes.
	ReconcilerActions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axisml_compute_reconciler_actions_total",
		Help: "Reconciler action outcomes.",
	}, []string{"resource", "predicate", "result"})

	// ReconcilerOldestPending exposes the oldest unprocessed row's age.
	ReconcilerOldestPending = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "axisml_compute_reconciler_oldest_pending_seconds",
		Help: "Age of the oldest reconciler work-set row.",
	}, []string{"resource", "predicate"})

	// SpecSyncPending is the count of rows with generation != observed_generation
	// (services_sync_pending partial index).
	SpecSyncPending = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "axisml_compute_spec_sync_pending_total",
		Help: "Rows with pending spec sync (generation != observed_generation).",
	}, []string{"resource"})

	// QuotaPrecheckRejected counts API-layer quota precheck rejections.
	QuotaPrecheckRejected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axisml_compute_quota_precheck_rejected_total",
		Help: "Quota precheck rejections.",
	}, []string{"tenant", "quota"})

	// CRDriftRepair counts CR drift repair events.
	CRDriftRepair = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axisml_compute_cr_drift_repair_total",
		Help: "CR drift repair events.",
	}, []string{"resource", "kind"})

	// InformerWorkqueueDepth is per-resource work queue depth.
	InformerWorkqueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "axisml_compute_informer_workqueue_depth",
		Help: "Per-module informer work queue depth.",
	}, []string{"resource"})
)

// Register installs all collectors into the controller-runtime metrics
// registry (which the manager also serves on /metrics).
func Register() {
	ctrlmetrics.Registry.MustRegister(
		IsLeader,
		HTTPRequestDuration,
		ReconcilerActions,
		ReconcilerOldestPending,
		SpecSyncPending,
		QuotaPrecheckRejected,
		CRDriftRepair,
		InformerWorkqueueDepth,
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
