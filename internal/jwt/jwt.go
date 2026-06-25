package jwt

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Data struct {
	SigningKey []byte
	CookieName string
	TokenTTL   time.Duration
	CookieTTL  time.Duration
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

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
