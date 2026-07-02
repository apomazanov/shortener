package jwt

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestData_CreateCookieWithUserID(t *testing.T) {
	signingKey := []byte("test-secret-key")
	cookieName := "auth_token"
	tokenTTL := 1 * time.Hour
	cookieTTL := 24 * time.Hour
	userID := "user-123"

	cfg := &Data{
		SigningKey: signingKey,
		CookieName: cookieName,
		TokenTTL:   tokenTTL,
		CookieTTL:  cookieTTL,
	}

	t.Run("success", func(t *testing.T) {
		cookie, err := cfg.CreateCookieWithUserID(userID)
		require.NoError(t, err)

		// Check cookie properties
		assert.Equal(t, cookieName, cookie.Name)
		assert.NotEmpty(t, cookie.Value)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.True(t, cookie.Expires.After(time.Now())) // Check if expiration is in the future

		// Parse the token to verify claims
		claims := Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, &claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrInvalidKeyType
			}
			return signingKey, nil
		})
		require.NoError(t, err)
		assert.True(t, token.Valid)
		assert.Equal(t, userID, claims.UserID)
		assert.True(t, claims.ExpiresAt.Time.After(time.Now())) // Check token expiration time
	})
}
