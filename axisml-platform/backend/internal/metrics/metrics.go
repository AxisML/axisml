package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registry is platform-backend's own collector registry. The backend runs no
// controller-runtime manager, so collectors are served directly via Handler.
var registry = prometheus.NewRegistry()

// Register installs the runtime collectors into the service registry. Call once
// at startup before Handler is served.
func Register() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// Handler serves the registered collectors in Prometheus text format. Mount it
// on the metrics listener (GET /metrics).
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
