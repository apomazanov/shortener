package audit

import (
	"context"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

// Subscriber defines methods for audit subscriber.
type Subscriber interface {
	Ch() chan<- domain.AuditEvent
}

// Dispatcher contains data for events dispatching activity.
type Dispatcher struct {
	// input is a channel for incoming events from AuditRecorder.
	input chan domain.AuditEvent
	// outputs is a slice of subscribers' channels.
	outputs []chan<- domain.AuditEvent
	// log is a pointer to system logger.
	log *zerolog.Logger
}

// NewDispatcher creates a new dispatcher object.
func NewDispatcher(log *zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		input:   make(chan domain.AuditEvent, 1000),
		outputs: make([]chan<- domain.AuditEvent, 0),
		log:     log,
	}
}

// InputChannel returns dispatcher input channel.
func (d *Dispatcher) InputChannel() chan<- domain.AuditEvent {
	return d.input
}

// Register registers a new subscriber for dispatcher.
func (d *Dispatcher) Register(sub Subscriber) {
	// thread safe, registering at application startup
	d.outputs = append(d.outputs, sub.Ch())
}

// Run provides continious dispatching of events.
func (d *Dispatcher) Run(ctx context.Context) {
	defer func() {
		for _, output := range d.outputs {
			close(output)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			d.log.Error().
				Err(ctx.Err()).
				Msg("audit: dispatcher failed stopping")
			return
		case event, ok := <-d.input:
			if !ok {
				return
			}
			for _, output := range d.outputs {
				select {
				case output <- event:
				case <-ctx.Done():
					d.log.Error().
						Err(ctx.Err()).
						Msg("audit: dispatcher failed stopping")
					return
				}

			}
		}
	}
}

// Stop stops dispatcher activity.
func (d *Dispatcher) Stop() {
	close(d.input)
}
