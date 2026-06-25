package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	my_jwt "github.com/apomazanov/shortener/internal/jwt"
)

type nextHandlerMock struct {
	called bool
	userID any
	err    error
}

func (m *nextHandlerMock) Handler(c *echo.Context) error {
	m.called = true
	m.userID = c.Get("user-id")
	return m.err
}

func makeAuthToken(t *testing.T, signingKey []byte, userID string, expiresAt time.Time) string {
	t.Helper()

	claims := my_jwt.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID: userID,
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(signingKey)
	require.NoError(t, err)

	return tokenString
}

func TestAuthenticator(t *testing.T) {
	jwtData := &my_jwt.Data{
		SigningKey: []byte("secret"),
		CookieName: "auth_token",
		TokenTTL:   time.Hour,
		CookieTTL:  24 * time.Hour,
	}

	log := zerolog.Nop()

	now := time.Now()

	tests := []struct {
		name              string
		method            string
		cookieValue       string
		nextErr           error
		expectedErr       error
		expectedNextCall  bool
		expectedNextUser  any
		expectedNewCookie bool
	}{
		{
			name:             "GET without cookie passes to next as anonymous user",
			method:           http.MethodGet,
			cookieValue:      "",
			expectedErr:      nil,
			expectedNextCall: true,
			expectedNextUser: nil,
		},
		{
			name:             "POST without cookie passes to next as anonymous user",
			method:           http.MethodPost,
			cookieValue:      "",
			expectedErr:      nil,
			expectedNextCall: true,
			expectedNextUser: nil,
		},
		{
			name:             "valid token sets user-id and calls next",
			method:           http.MethodGet,
			cookieValue:      makeAuthToken(t, jwtData.SigningKey, "user-1", now.Add(time.Hour)),
			expectedErr:      nil,
			expectedNextCall: true,
			expectedNextUser: "user-1",
		},
		{
			name:              "expired token is renewed and calls next",
			method:            http.MethodGet,
			cookieValue:       makeAuthToken(t, jwtData.SigningKey, "user-2", now.Add(-time.Hour)),
			expectedErr:       nil,
			expectedNextCall:  true,
			expectedNextUser:  "user-2",
			expectedNewCookie: true,
		},
		{
			name:             "garbage token passes to next as anonymous user",
			method:           http.MethodGet,
			cookieValue:      "not-a-jwt",
			expectedErr:      nil,
			expectedNextCall: true,
			expectedNextUser: nil,
		},
		{
			name:             "token signed with another key passes to next as anonymous user",
			method:           http.MethodGet,
			cookieValue:      makeAuthToken(t, []byte("another-secret"), "user-3", now.Add(time.Hour)),
			expectedErr:      nil,
			expectedNextCall: true,
			expectedNextUser: nil,
		},
		{
			name:             "next error is propagated",
			method:           http.MethodGet,
			cookieValue:      makeAuthToken(t, jwtData.SigningKey, "user-4", now.Add(time.Hour)),
			nextErr:          echo.ErrBadRequest,
			expectedErr:      echo.ErrBadRequest,
			expectedNextCall: true,
			expectedNextUser: "user-4",
		},
		{
			name:             "next error is propagated without cookie",
			method:           http.MethodPost,
			cookieValue:      "",
			nextErr:          errors.New("next handler error"),
			expectedErr:      errors.New("next handler error"),
			expectedNextCall: true,
			expectedNextUser: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  jwtData.CookieName,
					Value: tt.cookieValue,
				})
			}

			res := httptest.NewRecorder()
			ctx := e.NewContext(req, res)

			next := &nextHandlerMock{
				err: tt.nextErr,
			}

			handler := Authenticator(jwtData, &log)(next.Handler)

			err := handler(ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)

				if errors.Is(tt.expectedErr, echo.ErrBadRequest) {
					require.ErrorIs(t, err, tt.expectedErr)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedNextCall, next.called)
			assert.Equal(t, tt.expectedNextUser, next.userID)

			cookies := res.Result().Cookies()
			if tt.expectedNewCookie {
				require.Len(t, cookies, 1)

				renewedCookie := cookies[0]

				assert.Equal(t, jwtData.CookieName, renewedCookie.Name)
				assert.True(t, renewedCookie.HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, renewedCookie.SameSite)
				assert.True(t, renewedCookie.Expires.After(time.Now()))

				renewedClaims := my_jwt.Claims{}
				token, err := jwt.ParseWithClaims(renewedCookie.Value, &renewedClaims, func(token *jwt.Token) (any, error) {
					return jwtData.SigningKey, nil
				})

				require.NoError(t, err)
				require.True(t, token.Valid)
				assert.Equal(t, tt.expectedNextUser, renewedClaims.UserID)
				assert.True(t, renewedClaims.ExpiresAt.Time.After(time.Now()))
			} else {
				assert.Empty(t, cookies)
			}
		})
	}
}
