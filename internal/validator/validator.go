// Package validator contain methods for data validation.
package validator

import (
	"github.com/go-playground/validator/v10"
)

// MyValidator defines a validator object.
type MyValidator struct {
	// Validator is a pointer to validator.
	Validator *validator.Validate
}

// New creates and returns a new validator object.
func New() *MyValidator {
	return &MyValidator{Validator: validator.New()}
}

// Validate provides validation of provided object.
func (v *MyValidator) Validate(i any) error {
	return v.Validator.Struct(i)
}
