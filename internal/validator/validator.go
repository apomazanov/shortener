package validator

import (
	"github.com/go-playground/validator/v10"
)

type MyValidator struct {
	Validator *validator.Validate
}

/* -------------------------------------------------------------------------- */
func New() *MyValidator {
	return &MyValidator{Validator: validator.New()}
}

/* -------------------------------------------------------------------------- */
func (v *MyValidator) Validate(i any) error {
	return v.Validator.Struct(i)
}
