package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AuditToURL contains data for sending audit events to URL server.
type AuditToURL struct {
	// queue input channel with events queue.
	queue chan domain.AuditEvent
	// log is a pointer to system logger.
	log *zerolog.Logger
	// url is a remote URL address of events receiver.
	url string
	// httpClient is a http-client pointer for send operation.
	httpClient *http.Client
}

// NewAuditToURL creates a new audit-to-server subscriber object.
func NewAuditToURL(URL string, log *zerolog.Logger) *AuditToURL {

	retryClient := retryablehttp.NewClient()

	retryClient.RetryMax = 2
	retryClient.RetryWaitMin = 200 * time.Millisecond
	retryClient.RetryWaitMax = 1 * time.Second
	retryClient.HTTPClient.Timeout = 1 * time.Second
	retryClient.Logger = nil

	return &AuditToURL{
		queue:      make(chan domain.AuditEvent, 1000),
		log:        log,
		url:        URL,
		httpClient: retryClient.StandardClient(),
	}
}

// Ch returns subscriber's input channel.
func (s *AuditToURL) Ch() chan<- domain.AuditEvent {
	return s.queue
}

// Run provides continious queue monitoring and sending events to server.
func (s *AuditToURL) Run(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			s.log.Error().
				Err(ctx.Err()).
				Msg("audit: http: stopped by context")
			return
		case event, ok := <-s.queue:
			if !ok {
				return
			}
			err := s.send(ctx, event)
			if err != nil {
				log.Error().
					Err(err).
					Msg("audit: http: send error")
			}
		}
	}
}

// send provides send operation.
func (s *AuditToURL) send(ctx context.Context, event domain.AuditEvent) error {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 256*1024))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
