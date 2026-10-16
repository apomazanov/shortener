package middleware

import (
	"net/netip"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

// TrustedSubnetter provides subnet check for incoming requests.
func TrustedSubnetter(subnetStr string, log *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {

			ipStr := c.Request().Header.Get("X-Real-IP")

			if subnetStr == "" || ipStr == "" {
				// empty config or empty header don't need logging
				return echo.ErrForbidden
			}

			ip, err := netip.ParseAddr(ipStr)
			if err != nil {
				log.Warn().
					Err(err).
					Msg("trusted subnet: failed parsing ip")

				return echo.ErrForbidden
			}

			subnet, err := netip.ParsePrefix(subnetStr)
			if err != nil {
				log.Error().
					Err(err).
					Msg("trusted subnet: failed parsing config")

				return echo.ErrForbidden
			}

			if !subnet.Contains(ip) {
				return echo.ErrForbidden
			}

			return next(c)
		}
	}
}
