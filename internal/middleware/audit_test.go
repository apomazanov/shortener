package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apomazanov/shortener/internal/domain"
)

func TestAuditRecorder(t *testing.T) {
	log := zerolog.Nop()

	tests := []struct {
		name          string
		method        string
		userID        any
		original      any
		nextErr       error
		channelSize   int
		channelFull   bool
		expectedErr   error
		expectedEvent *domain.AuditEvent
	}{
		{
			name:        "GET request records follow event",
			method:      http.MethodGet,
			userID:      "user-1",
			original:    "https://example.com",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-1",
				URL:    "https://example.com",
				Action: "follow",
			},
		},
		{
			name:        "POST request records shorten event",
			method:      http.MethodPost,
			userID:      "user-2",
			original:    "https://example.com/long-url",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-2",
				URL:    "https://example.com/long-url",
				Action: "shorten",
			},
		},
		{
			name:        "PUT request records undefined action",
			method:      http.MethodPut,
			userID:      "user-3",
			original:    "https://example.com",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-3",
				URL:    "https://example.com",
				Action: "undefined",
			},
		},
		{
			name:        "DELETE request records undefined action",
			method:      http.MethodDelete,
			userID:      "user-4",
			original:    "https://example.com",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-4",
				URL:    "https://example.com",
				Action: "undefined",
			},
		},
		{
			name:        "missing user-id records empty string",
			method:      http.MethodGet,
			userID:      nil,
			original:    "https://example.com",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "",
				URL:    "https://example.com",
				Action: "follow",
			},
		},
		{
			name:        "missing original records empty string",
			method:      http.MethodGet,
			userID:      "user-5",
			original:    nil,
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-5",
				URL:    "",
				Action: "follow",
			},
		},
		{
			name:        "non-string user-id records empty string",
			method:      http.MethodGet,
			userID:      123,
			original:    "https://example.com",
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "",
				URL:    "https://example.com",
				Action: "follow",
			},
		},
		{
			name:        "non-string original records empty string",
			method:      http.MethodGet,
			userID:      "user-6",
			original:    456,
			channelSize: 1,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-6",
				URL:    "",
				Action: "follow",
			},
		},
		{
			name:          "next error skips event recording",
			method:        http.MethodGet,
			userID:        "user-7",
			original:      "https://example.com",
			nextErr:       errors.New("handler error"),
			channelSize:   1,
			expectedErr:   errors.New("handler error"),
			expectedEvent: nil,
		},
		{
			name:          "echo HTTP error skips event recording",
			method:        http.MethodPost,
			userID:        "user-8",
			original:      "https://example.com",
			nextErr:       echo.ErrBadRequest,
			channelSize:   1,
			expectedErr:   echo.ErrBadRequest,
			expectedEvent: nil,
		},
		{
			name:        "full channel drops event without blocking",
			method:      http.MethodGet,
			userID:      "user-9",
			original:    "https://example.com",
			channelSize: 1,
			channelFull: true,
			expectedEvent: &domain.AuditEvent{
				UserID: "user-9",
				URL:    "https://example.com",
				Action: "follow",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			req := httptest.NewRequest(tt.method, "/", nil)
			res := httptest.NewRecorder()
			ctx := e.NewContext(req, res)

			if tt.userID != nil {
				ctx.Set("user-id", tt.userID)
			}
			if tt.original != nil {
				ctx.Set("original", tt.original)
			}

			events := make(chan domain.AuditEvent, tt.channelSize)
			if tt.channelFull {
				events <- domain.AuditEvent{UserID: "existing", URL: "existing", Action: "existing"}
			}

			next := &nextHandlerMock{err: tt.nextErr}

			handler := AuditRecorder(events, &log)(next.Handler)

			before := time.Now().UnixMilli()
			err := handler(ctx)
			after := time.Now().UnixMilli()

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

			assert.True(t, next.called)

			if tt.expectedEvent == nil {
				assert.Empty(t, events)
				return
			}

			if tt.channelFull {
				// Channel was full, event should be dropped, existing event remains
				existing := <-events
				assert.Equal(t, "existing", existing.UserID)
				assert.Empty(t, events)
				return
			}

			require.Len(t, events, 1)

			event := <-events
			assert.Equal(t, tt.expectedEvent.UserID, event.UserID)
			assert.Equal(t, tt.expectedEvent.URL, event.URL)
			assert.Equal(t, tt.expectedEvent.Action, event.Action)
			assert.GreaterOrEqual(t, event.Timestamp, before)
			assert.LessOrEqual(t, event.Timestamp, after)
		})
	}
}
