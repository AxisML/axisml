package server

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/axisml/axisml/axisml-system/artifact-hub/pkg/strutil"
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
	if err := v.RegisterValidation("axisml_version", isVersion); err != nil {
		return err
	}
	return nil
}

func isAxisMLName(fl validator.FieldLevel) bool {
	return strutil.IsValidName(fl.Field().String())
}

func isVersion(fl validator.FieldLevel) bool {
	return strutil.IsValidVersion(fl.Field().String())
}
