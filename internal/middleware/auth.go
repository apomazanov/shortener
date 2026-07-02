package middleware

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"

	my_jwt "github.com/apomazanov/shortener/internal/jwt"
)

func Authenticator(jwtData *my_jwt.Data, log *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {

			// reading cookie

			cookie, err := c.Request().Cookie(jwtData.CookieName)
			cookieExists := (err == nil)

			if !cookieExists {
				c.Set("cookie-exists", false)
				return next(c)
			}
			c.Set("cookie-exists", true)

			// parsing token

			claims := my_jwt.Claims{}
			token, err := jwt.ParseWithClaims(cookie.Value, &claims, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
					return jwtData.SigningKey, nil
				}
				return nil, echo.ErrUnauthorized
			})

			// Token is OK

			if err == nil && token.Valid {
				c.Set("user-id", claims.UserID)
				return next(c)
			}

			// Token signature is OK but expired, renewing

			if errors.Is(err, jwt.ErrTokenExpired) {
				userID := claims.UserID

				newCookie, err := jwtData.CreateCookieWithUserID(userID)
				if err != nil {
					log.Error().
						Err(err).
						Msg("cookie creation failed")
					return echo.ErrInternalServerError
				}

				c.SetCookie(newCookie)
				c.Set("user-id", userID)
			}

			// If token is garbage, just skipping it

			return next(c)
		}
	}
}
