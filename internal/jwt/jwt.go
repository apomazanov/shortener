// Package jwt defines types and methods that help to operate with JWT tokens.
package jwt

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Data contains data stored in auth cookie.
type Data struct {
	// SigningKey is a signing key.
	SigningKey []byte
	// CookieName is a name of cookie.
	CookieName string
	// TokenTTL is a token time-to-live value.
	TokenTTL time.Duration
	// CookieTTL is a cookie time-to-live value.
	CookieTTL time.Duration
}

// Claims contains claims that are used in auth cookies.
type Claims struct {
	jwt.RegisteredClaims
	// UserID is a filed that contains request user-id.
	UserID string
}

// CreateCookieWithUserID creates a new cookie object for certain user.
func (d *Data) CreateCookieWithUserID(userID string) (*http.Cookie, error) {

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(d.TokenTTL)),
		},
		UserID: userID,
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	newTokenString, err := newToken.SignedString(d.SigningKey)
	if err != nil {
		return nil, fmt.Errorf("jwt: failed signing token: %w", err)
	}

	return &http.Cookie{
		Name:     d.CookieName,
		Value:    newTokenString,
		Expires:  time.Now().Add(d.CookieTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}, nil
}
