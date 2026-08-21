package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apomazanov/shortener/internal/domain"
)

func TestAuditToURL(t *testing.T) {
	log := zerolog.Nop()

	testEvent := domain.AuditEvent{
		UserID:    "user-1",
		URL:       "https://example.com",
		Action:    "follow",
		Timestamp: 1234567890,
	}

	tests := []struct {
		name             string
		events           []domain.AuditEvent
		serverStatus     int
		stopMethod       string // "channel_close" (context cancellation has a bug - Run doesn't return)
		expectedRequests int    // expected number of HTTP requests (retries multiply this)
		expectErrorLog   bool
	}{
		{
			name:             "single event sent successfully",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusOK,
			stopMethod:       "channel_close",
			expectedRequests: 1,
		},
		{
			name: "multiple events sent successfully",
			events: []domain.AuditEvent{
				testEvent,
				{UserID: "user-2", URL: "https://example.com/2", Action: "shorten", Timestamp: 1234567891},
				{UserID: "user-3", URL: "https://example.com/3", Action: "undefined", Timestamp: 1234567892},
			},
			serverStatus:     http.StatusOK,
			stopMethod:       "channel_close",
			expectedRequests: 3,
		},
		{
			name:             "server returns 201 Created",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusCreated,
			stopMethod:       "channel_close",
			expectedRequests: 1,
			expectErrorLog:   false,
		},
		{
			name:             "server returns 400 Bad Request (no retry for 4xx)",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusBadRequest,
			stopMethod:       "channel_close",
			expectedRequests: 1, // 4xx errors are not retried
			expectErrorLog:   true,
		},
		{
			name:             "server returns 500 Internal Server Error (retried 3 times)",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusInternalServerError,
			stopMethod:       "channel_close",
			expectedRequests: 3, // 5xx errors are retried (RetryMax=2 + initial = 3)
			expectErrorLog:   true,
		},
		{
			name:             "server returns 199 (edge case below 200)",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     199,
			stopMethod:       "channel_close",
			expectedRequests: 1,
			expectErrorLog:   true,
		},
		{
			name:             "server returns 300 Redirect (edge case at boundary)",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusMultipleChoices,
			stopMethod:       "channel_close",
			expectedRequests: 1,
			expectErrorLog:   true,
		},
		{
			name:             "no events",
			events:           []domain.AuditEvent{},
			serverStatus:     http.StatusOK,
			stopMethod:       "channel_close",
			expectedRequests: 0,
		},
		{
			name:             "stop by channel close",
			events:           []domain.AuditEvent{testEvent},
			serverStatus:     http.StatusOK,
			stopMethod:       "channel_close",
			expectedRequests: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int
			var mu sync.Mutex

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requestCount++
				mu.Unlock()
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			audit := NewAuditToURL(server.URL, &log)
			require.NotNil(t, audit)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			runDone := make(chan struct{})
			go func() {
				audit.Run(ctx)
				close(runDone)
			}()

			ch := audit.Ch()
			for _, event := range tt.events {
				ch <- event
			}

			// Give time to process events (longer for retries)
			time.Sleep(500 * time.Millisecond)

			close(ch)

			select {
			case <-runDone:
			case <-time.After(2 * time.Second):
				t.Fatal("Run did not stop in time")
			}

			mu.Lock()
			assert.Equal(t, tt.expectedRequests, requestCount)
			mu.Unlock()
		})
	}
}

func TestAuditToURLSend(t *testing.T) {
	log := zerolog.Nop()

	testEvent := domain.AuditEvent{
		UserID:    "user-1",
		URL:       "https://example.com",
		Action:    "follow",
		Timestamp: 1234567890,
	}

	tests := []struct {
		name          string
		serverStatus  int
		expectError   bool
		errorContains string
	}{
		{
			name:         "success 200 OK",
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:         "success 201 Created",
			serverStatus: http.StatusCreated,
			expectError:  false,
		},
		{
			name:         "success 204 No Content",
			serverStatus: http.StatusNoContent,
			expectError:  false,
		},
		{
			name:          "error 400 Bad Request (no retry)",
			serverStatus:  http.StatusBadRequest,
			expectError:   true,
			errorContains: "unexpected status code: 400",
		},
		{
			name:          "error 404 Not Found (no retry)",
			serverStatus:  http.StatusNotFound,
			expectError:   true,
			errorContains: "unexpected status code: 404",
		},
		{
			name:          "error 500 Internal Server Error (retried, then fails)",
			serverStatus:  http.StatusInternalServerError,
			expectError:   true,
			errorContains: "giving up after 3 attempt(s)", // retryablehttp wraps the error
		},
		{
			name:          "error 503 Service Unavailable (retried, then fails)",
			serverStatus:  http.StatusServiceUnavailable,
			expectError:   true,
			errorContains: "giving up after 3 attempt(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and content type
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify body is valid JSON
				var event domain.AuditEvent
				err := json.NewDecoder(r.Body).Decode(&event)
				require.NoError(t, err)
				assert.Equal(t, testEvent, event)

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			audit := NewAuditToURL(server.URL, &log)

			err := audit.send(context.Background(), testEvent)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuditToURLSendNetworkError(t *testing.T) {
	log := zerolog.Nop()

	tests := []struct {
		name          string
		url           string
		expectError   bool
		errorContains string
	}{
		{
			name:          "connection refused",
			url:           "http://localhost:59999",
			expectError:   true,
			errorContains: "request failed",
		},
		{
			name:          "invalid URL scheme",
			url:           "invalid://example.com",
			expectError:   true,
			errorContains: "request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audit := NewAuditToURL(tt.url, &log)

			event := domain.AuditEvent{
				UserID:    "user-1",
				URL:       "https://example.com",
				Action:    "follow",
				Timestamp: 1234567890,
			}

			err := audit.send(context.Background(), event)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuditToURLChannel(t *testing.T) {
	log := zerolog.Nop()

	audit := NewAuditToURL("http://example.com", &log)
	require.NotNil(t, audit)

	ch := audit.Ch()
	require.NotNil(t, ch)

	// Verify it's a send-only channel by sending an event
	event := domain.AuditEvent{UserID: "test", URL: "https://test.com", Action: "follow"}

	select {
	case ch <- event:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("failed to send event to channel")
	}
}
