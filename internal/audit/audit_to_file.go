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

type AuditToFile struct {
	queue   chan domain.AuditEvent
	log     *zerolog.Logger
	mu      sync.RWMutex
	file    *os.File
	encoder *json.Encoder
}

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

func (s *AuditToFile) Ch() chan<- domain.AuditEvent {
	return s.queue
}

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

func (s *AuditToFile) write(event domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encoder.Encode(event)
}

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
