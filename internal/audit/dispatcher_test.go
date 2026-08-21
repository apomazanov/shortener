package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apomazanov/shortener/internal/domain"
)

type subscriberMock struct {
	ch       chan domain.AuditEvent
	received []domain.AuditEvent
	mu       sync.Mutex
}

func newSubscriberMock(bufferSize int) *subscriberMock {
	return &subscriberMock{
		ch:       make(chan domain.AuditEvent, bufferSize),
		received: make([]domain.AuditEvent, 0),
	}
}

func (m *subscriberMock) Ch() chan<- domain.AuditEvent {
	return m.ch
}

func (m *subscriberMock) startCollecting(wg *sync.WaitGroup) {
	wg.Go(func() {
		for event := range m.ch {
			m.mu.Lock()
			m.received = append(m.received, event)
			m.mu.Unlock()
		}
	})
}

func (m *subscriberMock) getReceived() []domain.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.AuditEvent, len(m.received))
	copy(result, m.received)
	return result
}

func TestDispatcher(t *testing.T) {
	log := zerolog.Nop()

	testEvent1 := domain.AuditEvent{UserID: "user-1", URL: "https://example.com/1", Action: "follow", Timestamp: 1000}
	testEvent2 := domain.AuditEvent{UserID: "user-2", URL: "https://example.com/2", Action: "shorten", Timestamp: 2000}
	testEvent3 := domain.AuditEvent{UserID: "user-3", URL: "https://example.com/3", Action: "undefined", Timestamp: 3000}

	tests := []struct {
		name            string
		events          []domain.AuditEvent
		subscriberCount int
		stopMethod      string // "stop", "context", "input_close"
		expectedEvents  int
	}{
		{
			name:            "single event to single subscriber",
			events:          []domain.AuditEvent{testEvent1},
			subscriberCount: 1,
			stopMethod:      "stop",
			expectedEvents:  1,
		},
		{
			name:            "multiple events to single subscriber",
			events:          []domain.AuditEvent{testEvent1, testEvent2, testEvent3},
			subscriberCount: 1,
			stopMethod:      "stop",
			expectedEvents:  3,
		},
		{
			name:            "single event to multiple subscribers",
			events:          []domain.AuditEvent{testEvent1},
			subscriberCount: 3,
			stopMethod:      "stop",
			expectedEvents:  1,
		},
		{
			name:            "multiple events to multiple subscribers",
			events:          []domain.AuditEvent{testEvent1, testEvent2},
			subscriberCount: 2,
			stopMethod:      "stop",
			expectedEvents:  2,
		},
		{
			name:            "no events",
			events:          []domain.AuditEvent{},
			subscriberCount: 1,
			stopMethod:      "stop",
			expectedEvents:  0,
		},
		{
			name:            "no subscribers",
			events:          []domain.AuditEvent{testEvent1},
			subscriberCount: 0,
			stopMethod:      "stop",
			expectedEvents:  0,
		},
		{
			name:            "stop by context cancellation",
			events:          []domain.AuditEvent{testEvent1},
			subscriberCount: 1,
			stopMethod:      "context",
			expectedEvents:  1,
		},
		{
			name:            "stop by Stop method",
			events:          []domain.AuditEvent{testEvent1, testEvent2},
			subscriberCount: 1,
			stopMethod:      "stop",
			expectedEvents:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(&log)

			subscribers := make([]*subscriberMock, tt.subscriberCount)
			for i := range subscribers {
				subscribers[i] = newSubscriberMock(100)
				dispatcher.Register(subscribers[i])
			}

			var wg sync.WaitGroup
			for _, sub := range subscribers {
				sub.startCollecting(&wg)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runDone := make(chan struct{})
			go func() {
				dispatcher.Run(ctx)
				close(runDone)
			}()

			input := dispatcher.InputChannel()
			for _, event := range tt.events {
				input <- event
			}

			// Give dispatcher time to process events
			time.Sleep(50 * time.Millisecond)

			switch tt.stopMethod {
			case "context":
				cancel()
			case "stop":
				dispatcher.Stop()
			}

			select {
			case <-runDone:
			case <-time.After(2 * time.Second):
				t.Fatal("dispatcher did not stop in time")
			}

			// Wait for all subscribers to finish collecting
			wg.Wait()

			// Verify each subscriber received all events
			for i, sub := range subscribers {
				received := sub.getReceived()
				assert.Len(t, received, tt.expectedEvents, "subscriber %d", i)

				for j, event := range tt.events {
					if j < len(received) {
						assert.Equal(t, event, received[j], "subscriber %d event %d", i, j)
					}
				}
			}
		})
	}
}

func TestDispatcherOutputChannelsClosed(t *testing.T) {
	log := zerolog.Nop()

	tests := []struct {
		name            string
		subscriberCount int
		stopMethod      string
	}{
		{
			name:            "output channels closed after Stop",
			subscriberCount: 2,
			stopMethod:      "stop",
		},
		{
			name:            "output channels closed after context cancel",
			subscriberCount: 2,
			stopMethod:      "context",
		},
		{
			name:            "no subscribers to close",
			subscriberCount: 0,
			stopMethod:      "stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(&log)

			channels := make([]chan domain.AuditEvent, tt.subscriberCount)
			for i := range channels {
				ch := make(chan domain.AuditEvent, 10)
				channels[i] = ch
				sub := &subscriberMock{ch: ch}
				dispatcher.Register(sub)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runDone := make(chan struct{})
			go func() {
				dispatcher.Run(ctx)
				close(runDone)
			}()

			time.Sleep(10 * time.Millisecond)

			switch tt.stopMethod {
			case "context":
				cancel()
			case "stop":
				dispatcher.Stop()
			}

			select {
			case <-runDone:
			case <-time.After(2 * time.Second):
				t.Fatal("dispatcher did not stop in time")
			}

			// Verify all output channels are closed
			for i, ch := range channels {
				select {
				case _, ok := <-ch:
					assert.False(t, ok, "channel %d should be closed", i)
				default:
					// Channel is empty and closed, try to receive with timeout
					select {
					case _, ok := <-ch:
						assert.False(t, ok, "channel %d should be closed", i)
					case <-time.After(100 * time.Millisecond):
						t.Errorf("channel %d is not closed", i)
					}
				}
			}
		})
	}
}

func TestDispatcherInputChannel(t *testing.T) {
	log := zerolog.Nop()

	dispatcher := NewDispatcher(&log)

	input := dispatcher.InputChannel()
	require.NotNil(t, input)

	// Verify it's a send-only channel by sending an event
	event := domain.AuditEvent{UserID: "test", URL: "https://test.com", Action: "follow"}

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		dispatcher.Run(ctx)
		close(runDone)
	}()

	// Should not block
	select {
	case input <- event:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("failed to send event to input channel")
	}

	cancel()
	<-runDone
}
