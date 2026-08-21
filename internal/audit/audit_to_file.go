package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AuditToFile contains data for writing audit events to file.
type AuditToFile struct {
	// queue input channel with events queue.
	queue chan domain.AuditEvent
	// log is a pointer to system logger.
	log *zerolog.Logger
	// mu provides thread-safety during write operation.
	mu sync.RWMutex
	// file is a pointer to File object.
	file *os.File
	// encoder provides easier write operation to file.
	encoder *json.Encoder
}

// NewAuditToFile creates a new audit-to-file subscriber object.
func NewAuditToFile(fileName string, log *zerolog.Logger) *AuditToFile {

	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error().
			Err(err).
			Msg("audit: file: failed creating directory")
		return nil
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Error().
			Err(err).
			Msg("audit: file: failed create/open file")
		return nil
	}

	return &AuditToFile{
		queue:   make(chan domain.AuditEvent, 1000),
		log:     log,
		file:    file,
		encoder: json.NewEncoder(file),
	}
}

// Ch returns subscriber's input channel.
func (s *AuditToFile) Ch() chan<- domain.AuditEvent {
	return s.queue
}

// Run provides continious queue monitoring and writing events to file.
func (s *AuditToFile) Run(ctx context.Context) {
	defer s.close()

	for {
		select {
		case <-ctx.Done():
			s.log.Error().
				Err(ctx.Err()).
				Msg("audit: file: stopped by context")
			return
		case event, ok := <-s.queue:
			if !ok {
				return
			}
			err := s.write(event)
			if err != nil {
				log.Error().
					Err(err).
					Msg("audit: file: write error")
			}
		}
	}
}

// write provides write operation.
func (s *AuditToFile) write(event domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encoder.Encode(event)
}

// close correctly finishes write operation.
func (s *AuditToFile) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	errSync := s.file.Sync()
	errClose := s.file.Close()

	if errSync != nil {
		log.Error().
			Err(errSync).
			Msg("audit: file: sync error")
	}

	if errClose != nil {
		log.Error().
			Err(errClose).
			Msg("audit: file: close error")
	}
}
