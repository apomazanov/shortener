package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

// Repo interface declares repository methods needed for service functionality.
//
//go:generate mockgen -destination=mocks/mock_repo.gen.go -package=mocks github.com/apomazanov/shortener/internal/service Repo
type Repo interface {
	Save(ctx context.Context, alias string, original string, userID string) (usedAlias string, err error)
	SaveBatch(ctx context.Context, toWrite map[string]string, userID string) (written map[string]string, err error)
	Get(ctx context.Context, alias string) (original string, err error)
	GetByUser(ctx context.Context, userID string) (data map[string]string, err error)
	DeleteBatch(ctx context.Context, batch map[string][]string) error
}

// Service defines main business-logic data object.
type Service struct {
	// repo contains repository methods needed for data operations.
	repo Repo
	// deleter is a pointer to async delete handler.
	deleter *asyncDeleter
}

// New function returns a pointer to a new Service object.
func New(repo Repo, asyncDeleterCfg *AsyncDeleterConfig, log *zerolog.Logger) *Service {
	return &Service{
		repo:    repo,
		deleter: newAsyncDeleter(repo, asyncDeleterCfg, log),
	}
}

// newAlias creates a new randomly generated alias.
func (s *Service) newAlias() string {

	const (
		alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		aliasLen = 6
	)

	var buf [aliasLen]byte

	for i := range buf {
		buf[i] = alphabet[rand.IntN(len(alphabet))]
	}

	return string(buf[:])
}

// GetOriginalURL returns an original value of provided alias.
func (s *Service) GetOriginalURL(ctx context.Context, alias string) (original string, err error) {
	original, err = s.repo.Get(ctx, alias)

	if err != nil {
		return "", fmt.Errorf("service: cannot get original URL: %w", err)
	}

	return original, err
}

// GetUserURLs returns a map of aliases and their original values, which were
// created by provided user.
func (s *Service) GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error) {
	data, err = s.repo.GetByUser(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("service: cannot get user's URLs: %w", err)
	}

	return data, err
}

// CreateURLAlias creates a new alias for provided original URL. New value
// stored in repository, and new alias is returned.
func (s *Service) CreateURLAlias(ctx context.Context, original string, userID string) (alias string, err error) {
	const maxRetries = 5

	for range maxRetries {
		newAlias := s.newAlias()

		alias, err = s.repo.Save(ctx, newAlias, original, userID)

		if err == nil || errors.Is(err, domain.ErrOriginalURLDuplicate) {
			return alias, err
		}

		if errors.Is(err, domain.ErrAliasDuplicate) {
			// error is alias collision, keep trying
			continue
		}

		// error is not dublicate, smth wrong
		return "", fmt.Errorf("service: cannot create URL alias: %w", err)
	}

	return "", fmt.Errorf("service: alias gen retry limit exceeded: %w", domain.ErrSaveRetryLimitExceeded)
}

// CreateURLAliasBatch creates a batch of aliases for provided batch of original
// URLs. Original values are stored in repository, and batch of aliases is
// returned.
func (s *Service) CreateURLAliasBatch(ctx context.Context, originals []string, userID string) (writtenData map[string]string, err error) {
	const maxRetries = 5

	// original -> alias
	dataToWrite := make(map[string]string)

	for range maxRetries {

		for _, original := range originals {
			alias := s.newAlias()
			dataToWrite[original] = alias
		}

		writtenData, err := s.repo.SaveBatch(ctx, dataToWrite, userID)

		if err == nil || errors.Is(err, domain.ErrOriginalURLDuplicate) {
			return writtenData, err
		}

		if errors.Is(err, domain.ErrAliasDuplicate) {
			// if alias duplicate met, generating new aliases for full batch
			clear(dataToWrite)
			continue
		}

		// error is not dublicate, smth wrong
		return nil, fmt.Errorf("service: cannot create batch of URL aliases: %w", err)
	}

	return nil, fmt.Errorf("service: batch of aliases gen retry limit exceeded: %w", domain.ErrSaveRetryLimitExceeded)
}

// DeleteUserURLs deletes all alias-original pairs created by provided user.
// Delete operation is provided in async mode and can take a while.
func (s *Service) DeleteUserURLs(ctx context.Context, userID string, aliases []string) error {
	// ctx will be cancelled after response, not transiting it further
	s.deleter.Enqueue(userID, aliases)
	return nil
}
