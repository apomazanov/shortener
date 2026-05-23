package logger

import (
	"os"

	"github.com/rs/zerolog"
)

func New() *zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	l := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &l
}
