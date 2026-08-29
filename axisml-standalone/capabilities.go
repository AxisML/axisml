package standalone

import (
	"net/http"

	"github.com/gin-gonic/gin"

	arthubmodule "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/module"
	clustermodule "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/module"
	computemodule "github.com/axisml/axisml/axisml-system/compute-service/pkg/module"
)

// Capabilities is the System capability document axisml-standalone serves at
// GET /api/v1/capabilities. axisml-standalone composes all three modules into one
// process, so it aggregates each module's per-form capability document (derived
// from the injected providers) under its component key. Kubernetes serves
// each module's document from its own service instead; Platform forwards this
// verbatim (design §5.5).
type Capabilities struct {
	Components map[string]any `json:"components" desc:"Per-component capability documents, keyed by component name (cluster-manager / compute-service / artifact-hub)."`
}

// aggregateCapabilities folds the three modules' capability documents into the
// single standalone aggregate. Each module reports what its injected providers support
// (standalone wires read-only stores and the Docker runtime), so the document is
// truthful without restating any values here.
func aggregateCapabilities(
	clusterMod *clustermodule.Module,
	computeMod *computemodule.Module,
	arthubMod *arthubmodule.Module,
) Capabilities {
	return Capabilities{Components: map[string]any{
		"cluster-manager": clusterMod.Capabilities(),
		"compute-service": computeMod.Capabilities(),
		"artifact-hub":    arthubMod.Capabilities(),
	}}
}

// capabilitiesHandler serves the aggregate document unauthenticated so Platform
// can read it pre-login.
func capabilitiesHandler(caps Capabilities) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, caps) }
}
