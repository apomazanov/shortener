package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/apomazanov/shortener/internal/domain"
)

//go:generate mockgen -destination=mocks/mock_repo.go -package=mocks github.com/apomazanov/shortener/internal/service Repo
type Repo interface {
	Save(ctx context.Context, alias string, original string, userID string) (usedAlias string, err error)
	SaveBatch(ctx context.Context, toWrite map[string]string, userID string) (written map[string]string, err error)
	Get(ctx context.Context, alias string) (original string, err error)
	GetByUser(ctx context.Context, userID string) (data map[string]string, err error)
}

type Service struct {
	repo Repo
}

func New(repo Repo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) newAlias() string {

	const (
		alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		aliasLen = 6
	)

	var alias strings.Builder
	alias.Grow(aliasLen)

	for range aliasLen {
		idx := rand.IntN(len(alphabet))
		alias.WriteByte(alphabet[idx])
	}

	return alias.String()
}

func (s *Service) GetOriginalURL(ctx context.Context, alias string) (original string, err error) {
	original, err = s.repo.Get(ctx, alias)

	if err != nil {
		return "", fmt.Errorf("service: cannot get original URL: %w", err)
	}

	return original, err
}

func (s *Service) GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error) {
	data, err = s.repo.GetByUser(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("service: cannot get user's URLs: %w", err)
	}

	return data, err
}

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
