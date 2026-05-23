package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	v := New()
	assert.NotNil(t, v)
	assert.NotNil(t, v.Validator)
}

func TestValidator_Validate(t *testing.T) {
	v := New()

	type testStruct struct {
		Url string `validate:"required,url"`
	}

	t.Run("valid url", func(t *testing.T) {
		data := testStruct{Url: "https://google.com"}
		err := v.Validate(data)
		assert.NoError(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		data := testStruct{Url: "invalid-url"}
		err := v.Validate(data)
		assert.Error(t, err)
	})

	t.Run("empty url", func(t *testing.T) {
		data := testStruct{Url: ""}
		err := v.Validate(data)
		assert.Error(t, err)
	})
}
