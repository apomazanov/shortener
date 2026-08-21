package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apomazanov/shortener/internal/domain"
)

func TestNewAuditToFile(t *testing.T) {
	log := zerolog.Nop()

	tests := []struct {
		name       string
		fileName   string
		expectNil  bool
		expectFile bool
	}{
		{
			name:       "creates file in temp directory",
			fileName:   filepath.Join(t.TempDir(), "audit.log"),
			expectNil:  false,
			expectFile: true,
		},
		{
			name:       "creates nested directories",
			fileName:   filepath.Join(t.TempDir(), "nested", "deep", "audit.log"),
			expectNil:  false,
			expectFile: true,
		},
		{
			name:       "opens existing file for append",
			fileName:   filepath.Join(t.TempDir(), "existing.log"),
			expectNil:  false,
			expectFile: true,
		},
		{
			name:      "fails on invalid path",
			fileName:  "/dev/null/impossible/path/audit.log",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectFile {
				// For "opens existing file" test, pre-create the file
				if tt.name == "opens existing file for append" {
					err := os.WriteFile(tt.fileName, []byte("existing content\n"), 0o666)
					require.NoError(t, err)
				}
			}

			audit := NewAuditToFile(tt.fileName, &log)

			if tt.expectNil {
				assert.Nil(t, audit)
				return
			}

			require.NotNil(t, audit)
			assert.NotNil(t, audit.queue)
			assert.NotNil(t, audit.file)
			assert.NotNil(t, audit.encoder)

			info, err := os.Stat(tt.fileName)
			require.NoError(t, err)
			assert.False(t, info.IsDir())

			// Clean up: close the file handle
			audit.file.Close()
		})
	}
}

func TestAuditToFileRun(t *testing.T) {
	log := zerolog.Nop()

	testEvent1 := domain.AuditEvent{UserID: "user-1", URL: "https://example.com/1", Action: "follow", Timestamp: 1000}
	testEvent2 := domain.AuditEvent{UserID: "user-2", URL: "https://example.com/2", Action: "shorten", Timestamp: 2000}
	testEvent3 := domain.AuditEvent{UserID: "user-3", URL: "https://example.com/3", Action: "undefined", Timestamp: 3000}

	tests := []struct {
		name           string
		events         []domain.AuditEvent
		stopMethod     string // "context", "channel_close"
		expectedEvents int
	}{
		{
			name:           "single event written",
			events:         []domain.AuditEvent{testEvent1},
			stopMethod:     "channel_close",
			expectedEvents: 1,
		},
		{
			name:           "multiple events written in order",
			events:         []domain.AuditEvent{testEvent1, testEvent2, testEvent3},
			stopMethod:     "channel_close",
			expectedEvents: 3,
		},
		{
			name:           "no events",
			events:         []domain.AuditEvent{},
			stopMethod:     "channel_close",
			expectedEvents: 0,
		},
		{
			name:           "stop by context cancellation",
			events:         []domain.AuditEvent{testEvent1},
			stopMethod:     "context",
			expectedEvents: 1,
		},
		{
			name:           "stop by channel close",
			events:         []domain.AuditEvent{testEvent1, testEvent2},
			stopMethod:     "channel_close",
			expectedEvents: 2,
		},
		{
			name: "empty fields in event",
			events: []domain.AuditEvent{
				{UserID: "", URL: "", Action: "", Timestamp: 0},
			},
			stopMethod:     "channel_close",
			expectedEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := filepath.Join(t.TempDir(), "audit.log")

			audit := NewAuditToFile(fileName, &log)
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

			// Give time to process events
			time.Sleep(50 * time.Millisecond)

			switch tt.stopMethod {
			case "context":
				cancel()
			case "channel_close":
				close(ch)
			}

			select {
			case <-runDone:
			case <-time.After(2 * time.Second):
				t.Fatal("Run did not stop in time")
			}

			// Read and verify file contents
			content, err := os.ReadFile(fileName)
			require.NoError(t, err)

			if tt.expectedEvents == 0 {
				assert.Empty(t, content)
				return
			}

			// Each event is a JSON line (json.Encoder adds newline after each)
			lines := splitLines(string(content))
			assert.Len(t, lines, tt.expectedEvents)

			for i, expectedEvent := range tt.events {
				if i >= len(lines) {
					t.Errorf("missing event %d", i)
					continue
				}

				var event domain.AuditEvent
				err := json.Unmarshal([]byte(lines[i]), &event)
				require.NoError(t, err, "failed to parse line %d", i)
				assert.Equal(t, expectedEvent, event, "event %d mismatch", i)
			}
		})
	}
}

func TestAuditToFileAppendMode(t *testing.T) {
	log := zerolog.Nop()

	fileName := filepath.Join(t.TempDir(), "audit.log")

	// First run: write one event
	audit1 := NewAuditToFile(fileName, &log)
	require.NotNil(t, audit1)

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	runDone1 := make(chan struct{})
	go func() {
		audit1.Run(ctx1)
		close(runDone1)
	}()

	ch1 := audit1.Ch()
	ch1 <- domain.AuditEvent{UserID: "first", URL: "https://first.com", Action: "follow", Timestamp: 100}

	time.Sleep(50 * time.Millisecond)
	close(ch1)
	<-runDone1

	// Second run: write another event (should append)
	audit2 := NewAuditToFile(fileName, &log)
	require.NotNil(t, audit2)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	runDone2 := make(chan struct{})
	go func() {
		audit2.Run(ctx2)
		close(runDone2)
	}()

	ch2 := audit2.Ch()
	ch2 <- domain.AuditEvent{UserID: "second", URL: "https://second.com", Action: "shorten", Timestamp: 200}

	time.Sleep(50 * time.Millisecond)
	close(ch2)
	<-runDone2

	// Verify both events are in the file
	content, err := os.ReadFile(fileName)
	require.NoError(t, err)

	lines := splitLines(string(content))
	require.Len(t, lines, 2)

	var event1, event2 domain.AuditEvent
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &event1))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &event2))

	assert.Equal(t, "first", event1.UserID)
	assert.Equal(t, "second", event2.UserID)
}

func TestAuditToFileChannel(t *testing.T) {
	log := zerolog.Nop()

	fileName := filepath.Join(t.TempDir(), "audit.log")
	audit := NewAuditToFile(fileName, &log)
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

	// Clean up
	audit.file.Close()
}

func TestAuditToFileClose(t *testing.T) {
	log := zerolog.Nop()

	tests := []struct {
		name string
	}{
		{name: "close syncs and closes file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := filepath.Join(t.TempDir(), "audit.log")

			audit := NewAuditToFile(fileName, &log)
			require.NotNil(t, audit)

			// Write some data
			audit.write(domain.AuditEvent{UserID: "user-1", URL: "https://example.com", Action: "follow", Timestamp: 1000})

			// Close should sync and close the file
			audit.close()

			// File should be readable after close
			content, err := os.ReadFile(fileName)
			require.NoError(t, err)
			assert.NotEmpty(t, content)

			// Verify the content is valid JSON
			var event domain.AuditEvent
			require.NoError(t, json.Unmarshal(content[:len(content)-1], &event)) // strip trailing newline
			assert.Equal(t, "user-1", event.UserID)
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			line := s[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}
