package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedSubnetter(t *testing.T) {

	log := zerolog.Nop()

	tests := []struct {
		name                  string
		trustedSubnet         string
		xRealIPHeader         string
		expectedCreationErr   error
		expectedMiddlewareErr error
		expectedHandlerCalled bool
	}{
		{
			name:                  "access granted",
			trustedSubnet:         "192.168.1.0/24",
			xRealIPHeader:         "192.168.1.1",
			expectedCreationErr:   nil,
			expectedMiddlewareErr: nil,
			expectedHandlerCalled: true,
		},
		{
			name:                  "ip outside trusted subnet",
			trustedSubnet:         "192.168.1.0/24",
			xRealIPHeader:         "192.168.2.1",
			expectedCreationErr:   nil,
			expectedMiddlewareErr: echo.ErrForbidden,
			expectedHandlerCalled: false,
		},
		{
			name:                  "empty trusted subnet",
			trustedSubnet:         "",
			xRealIPHeader:         "192.168.1.1",
			expectedCreationErr:   echo.ErrForbidden,
			expectedMiddlewareErr: nil,
			expectedHandlerCalled: false,
		},
		{
			name:                  "empty header",
			trustedSubnet:         "192.168.1.0/24",
			xRealIPHeader:         "",
			expectedCreationErr:   nil,
			expectedMiddlewareErr: echo.ErrForbidden,
			expectedHandlerCalled: false,
		},
		{
			name:                  "invalid header value",
			trustedSubnet:         "192.168.1.0/24",
			xRealIPHeader:         "garbage",
			expectedCreationErr:   nil,
			expectedMiddlewareErr: echo.ErrForbidden,
			expectedHandlerCalled: false,
		},
		{
			name:                  "invalid trusted subnet value",
			trustedSubnet:         "garbage",
			xRealIPHeader:         "192.168.1.1",
			expectedCreationErr:   echo.ErrForbidden,
			expectedMiddlewareErr: nil,
			expectedHandlerCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xRealIPHeader != "" {
				req.Header.Set("X-Real-IP", tt.xRealIPHeader)
			}

			res := httptest.NewRecorder()
			ctx := e.NewContext(req, res)

			var handlerCalled bool

			next := &nextHandlerMock{
				handler: func(c *echo.Context) error {
					handlerCalled = true
					return nil
				},
			}

			subnetter, errCreate := TrustedSubnetter(tt.trustedSubnet, &log)
			handler := subnetter(next.Handler)
			errHandler := handler(ctx)

			if tt.expectedCreationErr != nil {
				require.Error(t, errCreate)
				require.Error(t, errHandler)
			} else {
				require.NoError(t, errCreate)

				if tt.expectedMiddlewareErr != nil {
					require.Error(t, errHandler)
				} else {
					require.NoError(t, errHandler)
				}
				assert.Equal(t, tt.expectedHandlerCalled, handlerCalled)
			}

		})
	}
}
