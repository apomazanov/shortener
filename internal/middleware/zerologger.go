package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

/* -------------------------------------------------------------------------- */
func getResponseData(c *echo.Context, e error) (status int, size int64) {
	// Trying to get actual data from ResponseWriter
	if obj, ok := c.Response().(*echo.Response); ok {
		status = obj.Status
		size = obj.Size
	}

	// Status in ResponseWriter might be invalid, checking handler error
	if e != nil {
		var he *echo.HTTPError
		if errors.As(e, &he) {
			status = he.Code
		} else {
			status = http.StatusInternalServerError
		}
	}

	// In case status is still 0
	if status <= 0 {
		status = http.StatusOK
	}

	return status, size
}

/* -------------------------------------------------------------------------- */
func Zerologger(l *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()

			log := l.With().
				Str("request_id", c.Response().Header().Get(echo.HeaderXRequestID)).
				Str("method", c.Request().Method).
				Str("uri", c.Request().RequestURI).
				Logger()

			err := next(c)

			status, size := getResponseData(c, err)

			log.Info().
				Dur("latency_ms", time.Since(start)).
				Int("status", status).
				Int64("response_size", size).
				Msg("")

			return err
		}
	}
}
