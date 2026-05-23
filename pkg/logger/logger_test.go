package logger

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

/* -------------------------------------------------------------------------- */
func TestNew(t *testing.T) {
	logger := New()

	assert.NotNil(t, logger)
	assert.Equal(t, zerolog.TimeFormatUnixMs, zerolog.TimeFieldFormat)
}
