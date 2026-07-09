package module_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/module"
)

func init() { gin.SetMode(gin.TestMode) }

func TestNew_DefaultsAndAssembly(t *testing.T) {
	m, err := module.New(module.Deps{Log: logr.Discard()})
	require.NoError(t, err)
	require.NotNil(t, m)

	// Three CRs (MLRun / MLService / MLTrafficPolicy) → three of each moving part.
	assert.Len(t, m.Routes(), 3)
	assert.Len(t, m.Runnables(), 3)
	assert.Len(t, m.StatusReflowRunnables(), 3)

	// RuntimeName defaults to "kubernetes"; QuotaEnforcement defaults to false.
	caps := m.Capabilities()
	assert.Equal(t, "kubernetes", caps.Runtime)
	assert.False(t, caps.QuotaEnforcement)
}

func TestNew_RuntimeOverride(t *testing.T) {
	m, err := module.New(module.Deps{
		Log:              logr.Discard(),
		RuntimeName:      "standalone",
		QuotaEnforcement: true,
	})
	require.NoError(t, err)
	caps := m.Capabilities()
	assert.Equal(t, "standalone", caps.Runtime)
	assert.True(t, caps.QuotaEnforcement)
}

func TestRegisterRoutes(t *testing.T) {
	m, err := module.New(module.Deps{Log: logr.Discard()})
	require.NoError(t, err)

	eng := gin.New()
	grp := eng.Group("/api/v1")
	assert.NotPanics(t, func() { m.RegisterRoutes(grp) })

	// Every route handler is mounted; the engine should now expose routes.
	assert.NotEmpty(t, eng.Routes())
}

func TestHTTPReExports(t *testing.T) {
	// RegisterValidators is idempotent and must succeed.
	require.NoError(t, module.RegisterValidators())
	require.NoError(t, module.RegisterValidators())

	// The middleware constructors return usable handlers (no external I/O).
	assert.NotNil(t, module.RequestID())
	assert.NotNil(t, module.AccessLog(logr.Discard()))
	assert.NotNil(t, module.Recovery(logr.Discard()))
	assert.NotNil(t, module.ErrorHandler())
	assert.NotNil(t, module.IdentityMiddleware())
}
