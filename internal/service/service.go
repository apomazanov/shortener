package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/apomazanov/shortener/internal/domain"
)

type Repo interface {
	Save(ctx context.Context, alias string, original string) error
	Get(ctx context.Context, alias string) (original string, err error)
}

type Service struct {
	repo Repo
}

/* -------------------------------------------------------------------------- */
func New(repo Repo) *Service {
	return &Service{
		repo: repo,
	}
}

/* -------------------------------------------------------------------------- */
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

/* -------------------------------------------------------------------------- */
func (s *Service) GetOriginalURL(ctx context.Context, alias string) (original string, err error) {
	original, err = s.repo.Get(ctx, alias)

	if err != nil {
		return "", fmt.Errorf("service: cannot get original URL: %w", err)
	}

	return original, err
}

/* -------------------------------------------------------------------------- */
func (s *Service) CreateURLAlias(ctx context.Context, original string) (alias string, err error) {
	const maxRetries = 5

	for range maxRetries {
		alias = s.newAlias()

		err = s.repo.Save(ctx, alias, original)

		if err == nil {
			return alias, err
		}

		if errors.Is(err, domain.ErrDuplicate) {
			// error is alias collision, keep trying
			continue
		}

		// error is not dublicate, smth wrong
		return "", fmt.Errorf("service: cannot create URL alias: %w", err)
	}

	return "", fmt.Errorf("service: alias gen retry limit exceeded: %w", domain.ErrSaveRetryLimitExceeded)
}
