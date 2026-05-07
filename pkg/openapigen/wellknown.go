package openapigen

import (
	"reflect"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// WellKnownFunc lets a per-service generator inject extra type → schema
// mappings (e.g. an apperrors.Code enum) on top of the built-in table.
// Returning nil falls through to the next mapper / default reflection path.
type WellKnownFunc func(reflect.Type) *Schema

// Type-identity comparisons avoid stringly-typed package-path checks. Pulled
// out so the switch in builtinWellKnown stays a flat type table.
var (
	timeType         = reflect.TypeOf(time.Time{})
	uuidType         = reflect.TypeOf(uuid.UUID{})
	quantityType     = reflect.TypeOf(resource.Quantity{})
	resourceListType = reflect.TypeOf(corev1.ResourceList{})
)

// builtinWellKnown maps Go types that don't have a useful struct-reflective
// shape (or that we want to render with a specific OpenAPI flavor) to a fixed
// Schema. Returning nil falls through to the default reflection path (or the
// caller's user-supplied WellKnownFunc).
func builtinWellKnown(t reflect.Type) *Schema {
	switch t {
	case timeType:
		return &Schema{Type: "string", Format: "date-time"}
	case uuidType:
		return &Schema{Type: "string", Format: "uuid"}
	case quantityType:
		return &Schema{
			Type:        "string",
			Description: "Kubernetes resource.Quantity (e.g. \"500m\", \"2Gi\", \"4\").",
		}
	case resourceListType:
		return &Schema{
			Type: "object",
			AdditionalProperties: &Schema{
				Type:        "string",
				Description: "Kubernetes resource.Quantity.",
			},
			Description: "Map of resource name (cpu, memory, nvidia.com/gpu, …) to resource.Quantity.",
		}
	}
	// metav1.Duration has a single String-marshalling Duration field; render it
	// as a string instead of an object.
	if t.Kind() == reflect.Struct && t.PkgPath() == "k8s.io/apimachinery/pkg/apis/meta/v1" && t.Name() == "Duration" {
		return &Schema{Type: "string", Description: "Go duration string (e.g. \"30s\", \"5m\")."}
	}
	// runtime.RawExtension exposes only json:"-" fields, so reflection yields
	// an empty object. Render it as free-form so clients know it's opaque.
	if t.Kind() == reflect.Struct && t.PkgPath() == "k8s.io/apimachinery/pkg/runtime" && t.Name() == "RawExtension" {
		return &Schema{Type: "object", Description: "Embedded Kubernetes resource (free-form JSON)."}
	}
	return nil
}
