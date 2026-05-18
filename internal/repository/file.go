package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

type FileRepo struct {
	file string
	mu   sync.RWMutex
	log  *zerolog.Logger
	cache map[string]string
	lastUuid int
}

type RepoConfig interface {
	GetRepoFile() string
}

type entry struct {
	Uuid     int `json:"uuid"`
	Alias    string `json:"alias"`
	Original string `json:"original"`
}

/* -------------------------------------------------------------------------- */
func NewFileRepo(cfg RepoConfig, l *zerolog.Logger) (*FileRepo, error) {

	repo := &FileRepo{file: cfg.GetRepoFile(), log: l, cache: make(map[string]string)}

	// Create storage file, if not exists

	file, err := os.OpenFile(repo.file, os.O_CREATE | os.O_RDONLY, 0o644)
	if err != nil {
		repo.log.Error().
			Str("op", "repository.NewFileRepo").
			Err(err).
			Msg("storage file creation failed")
		return nil, err
	}
	defer file.Close()

	// Read existing file (empty, if new) to fill aliases cache

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var e entry
		if err = json.Unmarshal(line, &e); err != nil {
			repo.log.Warn().
				Str("op", "repository.NewFileRepo").
				Str("string", string(line)).
				Err(err).
				Msg(fmt.Sprintf("JSON unmarshal error at line %d", lineNum))
			// bad record is just passed by
			continue
		}

		repo.cache[e.Alias] = e.Original

		if repo.lastUuid < e.Uuid {
			repo.lastUuid = e.Uuid
		}
	}

	if err := scanner.Err(); err != nil {
		repo.log.Error().
			Str("op", "repository.NewFileRepo").
			Err(err).
			Msg("reading storage file failed")

		return nil, err
	}

	return repo, nil
}

/* -------------------------------------------------------------------------- */
func (r *FileRepo) Save(ctx context.Context, alias string, original string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Searching for duplicates first

	if _, exists := r.cache[alias]; exists {
		return domain.ErrDuplicate
	}

	// Appending

	// Opening file
	f, err := os.OpenFile(r.file, os.O_APPEND | os.O_WRONLY, 0o644)
	if err != nil {
		r.log.Error().
			Str("op", "repository.Save").
			Err(err).
			Msg("cannot open file")

		return err
	}
	defer f.Close()

	// Preparing data
	newLastUuid := r.lastUuid + 1
	entry := entry{Uuid: newLastUuid, Alias: alias, Original: original}
	data, err := json.Marshal(entry)
	if err != nil {
		r.log.Error().
			Str("op", "repository.Save").
			Err(err).
			Msg("JSON marshal error")

		return err
	}
	data = append(data, '\n')

	// Writing data
	_, err = f.Write(data)
	if err != nil {
		r.log.Error().
			Str("op", "repository.Save").
			Err(err).
			Msg("writing to file failed")

		return err
	}

	// Updating cache, if data write was successful
	r.cache[alias] = original

	return nil
}

/* -------------------------------------------------------------------------- */
func (r *FileRepo) Get(ctx context.Context, alias string) (original string, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	original, exists := r.cache[alias]
	if exists {
		return original, nil
	}

	return "", domain.ErrNotFound
}
