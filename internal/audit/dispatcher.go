package audit

import (
	"context"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

type Subscriber interface {
	Ch() chan<- domain.AuditEvent
}

type Dispatcher struct {
	input   chan domain.AuditEvent
	outputs []chan<- domain.AuditEvent
	log     *zerolog.Logger
}

func NewDispatcher(log *zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		input:   make(chan domain.AuditEvent, 1000),
		outputs: make([]chan<- domain.AuditEvent, 0),
		log:     log,
	}
}

func (d *Dispatcher) InputChannel() chan<- domain.AuditEvent {
	return d.input
}

func (d *Dispatcher) Register(sub Subscriber) {
	// thread safe, registering at application startup
	d.outputs = append(d.outputs, sub.Ch())
}

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

func (d *Dispatcher) Stop() {
	close(d.input)
}
