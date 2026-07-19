package server

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/strutil"
)

// RegisterValidators wires AxisML-specific tags into the gin binding engine.
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	if err := v.RegisterValidation("axisml_name", isAxisMLName); err != nil {
		return err
	}
	if err := v.RegisterValidation("axisml_resource_unit", isResourceUnitName); err != nil {
		return err
	}
	if err := v.RegisterValidation("configmap_name", isConfigMapName); err != nil {
		return err
	}
	return nil
}

func isAxisMLName(fl validator.FieldLevel) bool {
	return strutil.IsValidName(fl.Field().String())
}

func isResourceUnitName(fl validator.FieldLevel) bool {
	return strutil.IsValidResourceUnitName(fl.Field().String())
}

func isConfigMapName(fl validator.FieldLevel) bool {
	return len(k8svalidation.IsDNS1123Subdomain(fl.Field().String())) == 0
}
