package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

type Service interface {
	GetOriginalUrl(ctx context.Context, alias string) (original string, err error)
	CreateUrlAlias(ctx context.Context, original string) (alias string, err error)
}

type UrlConfig interface {
	GetUrlBase() string
	GetAliasSize() int
}

type Handler struct {
	srv Service
	cfg UrlConfig
	log *zerolog.Logger
}

/* -------------------------------------------------------------------------- */
func New(s Service, c UrlConfig, l *zerolog.Logger) *Handler {
	return &Handler{
		srv: s,
		cfg: c,
		log: l,
	}
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Get(c *echo.Context) error {

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Getting alias from request URL
	alias := c.Param("short")

	if len(alias) != h.cfg.GetAliasSize() {
		return echo.NewHTTPError(http.StatusBadRequest, "alias is invalid")
	}

	// Find original URL
	original, err := h.srv.GetOriginalUrl(ctx, alias)

	if err == nil {
		// Return original URL with redirection
		return c.Redirect(http.StatusTemporaryRedirect, original)
	}

	if errors.Is(err, domain.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "URL not found")
	}

	return echo.NewHTTPError(http.StatusInternalServerError, "Something went wrong")
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Create(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.Create").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body, expecting long URL for shortening
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to read request body")

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Casting body to string, validating
	original := string(body)
	if original == "" {
		log.Warn().
			Msg("no URL to shorten in request")

		return echo.NewHTTPError(http.StatusBadRequest, "No URL passed")
	}

	// Getting alias
	alias, err := h.srv.CreateUrlAlias(ctx, original)
	if err != nil {
		// logged in service layer
		return echo.NewHTTPError(http.StatusInternalServerError, "Aliasing failed")
	}

	// Sending back short URL
	shortUrl, _ := url.JoinPath(h.cfg.GetUrlBase(), alias)
	return c.String(http.StatusCreated, shortUrl)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.NewHTTPError(http.StatusBadRequest, "")
}
