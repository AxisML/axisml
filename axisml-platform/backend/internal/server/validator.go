package server

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Patterns mirror cmd/openapi-gen (the generated spec's path/field patterns).
var (
	dns1123Re       = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	configMapNameRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	artifactNameRe  = regexp.MustCompile(`^[a-z0-9]([-a-z0-9._]*[a-z0-9])?$`)
)

// RegisterValidators wires AxisML-specific binding tags. Tags must match the
// `binding:"..."` tags used by the request/response types (dns1123 / artifactname).
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	if err := v.RegisterValidation("dns1123", func(fl validator.FieldLevel) bool {
		return dns1123Re.MatchString(fl.Field().String())
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("artifactname", func(fl validator.FieldLevel) bool {
		return artifactNameRe.MatchString(fl.Field().String())
	}); err != nil {
		return err
	}
	if err := v.RegisterValidation("configmap_name", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return len(value) <= 253 && configMapNameRe.MatchString(value)
	}); err != nil {
		return err
	}
	return nil
}
