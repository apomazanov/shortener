package middleware

import (
	"net/http"
	"time"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

// AuditRecorder provides requests monitoring for audit service. It creates
// audit events and sends them to audit manager.
func AuditRecorder(events chan<- domain.AuditEvent, log *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {

			err := next(c)

			if err == nil {
				event := domain.AuditEvent{
					UserID:    extractStringFromCtx(c, "user-id"),
					URL:       extractStringFromCtx(c, "original"),
					Timestamp: time.Now().UnixMilli(),
				}

				switch c.Request().Method {
				case http.MethodGet:
					event.Action = "follow"
				case http.MethodPost:
					event.Action = "shorten"
				default:
					event.Action = "undefined"
				}

				select {
				case events <- event:
				default:
					// skipping in case of overload
					log.Warn().
						Msg("audit: recorder overload")
				}
			}

			return err
		}
	}
}

// extractStringFromCtx is a helper for extracting string values from context
// by key.
func extractStringFromCtx(c *echo.Context, s string) string {

	valueAny := c.Get(s)

	if valueAny == nil {
		return ""
	}

	value, _ := valueAny.(string)
	return value
}
