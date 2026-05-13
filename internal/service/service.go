package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/rs/zerolog"
)

type Repo interface {
	Save(ctx context.Context, alias string, original string) error
	Get(ctx context.Context, alias string) (original string, err error)
}

type Service struct {
	repo Repo
	log  *zerolog.Logger
}

/* -------------------------------------------------------------------------- */
func New(repo Repo, log *zerolog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

/* -------------------------------------------------------------------------- */
func (s *Service) newAlias() (alias string) {

	const (
		alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		aliasLen = 6
	)

	var res strings.Builder
	res.Grow(aliasLen)

	for range aliasLen {
		idx := rand.IntN(len(alphabet))
		res.WriteByte(alphabet[idx])
	}

	alias = res.String()

	s.log.Debug().
		Str("op", "service.newShort").
		Str("created_alias", alias).
		Msg("new alias generated")

	return alias
}

/* -------------------------------------------------------------------------- */
func (s *Service) GetOriginalUrl(ctx context.Context, alias string) (original string, err error) {
	original, err = s.repo.Get(ctx, alias)

	if err == nil {
		return original, err
	}

	if !errors.Is(err, domain.ErrNotFound) {
		s.log.Error().
			Str("op", "service.GetOriginalUrl").
			Err(err).
			Str("alias", alias).
			Msg("alias lookup failed")
	}

	return "", err
}

/* -------------------------------------------------------------------------- */
func (s *Service) CreateUrlAlias(ctx context.Context, original string) (alias string, err error) {
	const maxRetries = 5

	log := s.log.With().
		Str("op", "service.CreateUrlAlias").
		Str("original_url", original).
		Logger()

	for range maxRetries {
		alias = s.newAlias()

		err = s.repo.Save(ctx, alias, original)

		if err == nil {
			return alias, err
		}

		// error is not dublicate, smth wrong
		if !errors.Is(err, domain.ErrDuplicate) {
			// logged in repo layer
			return "", err
		}

		// error is alias collision, keep trying
	}

	log.Error().
		Err(err).
		Int("retries", maxRetries).
		Msg("failed to generate unique alias: retry limit exceeded")

	return "", domain.ErrSaveRetryLimitExceeded
}
