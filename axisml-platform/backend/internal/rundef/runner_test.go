package rundef

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

func TestEffectiveAnnotationsOnlyInheritsPriority(t *testing.T) {
	definition := map[string]string{
		server.MLRunPriorityAnnotation: "10",
		"example.com/note":             "definition-only",
	}
	got := effectiveAnnotations(definition, &server.RunTriggerRequest{Annotations: server.StringMap{
		"example.com/trigger": "kept",
	}})
	assert.Equal(t, map[string]string{
		server.MLRunPriorityAnnotation: "10",
		"example.com/trigger":          "kept",
	}, got)
}

func TestEffectiveAnnotationsTriggerPriorityWins(t *testing.T) {
	got := effectiveAnnotations(
		map[string]string{server.MLRunPriorityAnnotation: "10"},
		&server.RunTriggerRequest{Annotations: server.StringMap{server.MLRunPriorityAnnotation: "20"}},
	)
	assert.Equal(t, "20", got[server.MLRunPriorityAnnotation])
}

func TestValidatePriorityAnnotations(t *testing.T) {
	assert.NoError(t, ValidatePriorityAnnotations(map[string]string{server.MLRunPriorityAnnotation: "-2147483648"}))
	err := ValidatePriorityAnnotations(map[string]string{server.MLRunPriorityAnnotation: "2147483648"})
	assert.Error(t, err)
}
