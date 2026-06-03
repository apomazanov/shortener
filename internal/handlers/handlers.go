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

const aliasSize = 6

type BusinessService interface {
	GetOriginalURL(ctx context.Context, alias string) (original string, err error)
	CreateURLAlias(ctx context.Context, original string) (alias string, err error)
}

type HealthService interface {
	Ping(ctx context.Context) error
}

type URLConfig interface {
	GetURLBase() string
}

type Handler struct {
	business BusinessService
	health   HealthService
	cfg      URLConfig
	log      *zerolog.Logger
}

type jsonShortenRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type jsonShortenResponse struct {
	Result string `json:"result"`
}

type jsonBatchRequestItem struct {
	ID       string `json:"correlation_id" validate:"required"`
	Original string `json:"original_url" validate:"required,url"`
}

type jsonBatchResponseItem struct {
	ID     string `json:"correlation_id"`
	Result string `json:"short_url"`
}

/* -------------------------------------------------------------------------- */
func New(b BusinessService, c URLConfig, l *zerolog.Logger, h HealthService) *Handler {
	return &Handler{
		business: b,
		health:   h,
		cfg:      c,
		log:      l,
	}
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Get(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.Get").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Getting alias from request URL
	alias := c.Param("alias")

	if len(alias) != aliasSize {
		log.Info().
			Str("alias", alias).
			Msg("invalid alias")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Find original URL
	original, err := h.business.GetOriginalURL(ctx, alias)

	if err == nil {
		// Return original URL with redirection
		return c.Redirect(http.StatusTemporaryRedirect, original)
	}

	if errors.Is(err, domain.ErrNotFound) {
		log.Info().
			Err(err).
			Msg("alias not found")

		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	log.Error().
		Err(err).
		Msg("something went wrong")

	return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Ping(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.Ping").Logger()

	ctx := c.Request().Context()

	if err := h.health.Ping(ctx); err != nil {

		log.Error().
			Err(err).
			Msg("health check failed")

		return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return c.NoContent(http.StatusOK)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) CreateText(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.CreateText").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body, expecting long URL for shortening
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Casting body to string, validating
	requestData := jsonShortenRequest{URL: string(body)}
	if err := c.Validate(&requestData); err != nil {
		log.Info().
			Err(err).
			Any("request_data", requestData).
			Msg("invalid request data")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL)
	if err != nil {
		log.Error().
			Err(err).
			Str("alias", alias).
			Msg("alias creation failed")

		return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	// Sending back short URL

	// base URL already validated during config
	shortUrl := h.cfg.GetURLBase() + "/" + alias
	return c.String(http.StatusCreated, shortUrl)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) CreateJson(c *echo.Context) error {
	var requestData jsonShortenRequest
	var responseData jsonShortenResponse

	log := h.log.With().Str("op", "handler.CreateJson").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body
	if err := c.Bind(&requestData); err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Validating
	if err := c.Validate(&requestData); err != nil {
		log.Info().
			Err(err).
			Any("request_data", requestData).
			Msg("invalid request data")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL)
	if err != nil {
		log.Error().
			Err(err).
			Str("alias", alias).
			Msg("alias creation failed")

		return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	// Sending back short URL
	responseData.Result, _ = url.JoinPath(h.cfg.GetURLBase(), alias)
	return c.JSON(http.StatusCreated, responseData)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) CreateJsonBatch(c *echo.Context) error {
	var requestData []jsonBatchRequestItem
	var responseData []jsonBatchResponseItem

	log := h.log.With().Str("op", "handler.CreateJsonBatch").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body
	if err := c.Bind(&requestData); err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	}

	// Processing each item
	for _, item := range requestData {
		// Validating
		if err := c.Validate(&item); err != nil {
			log.Info().
				Err(err).
				Any("request_item", item).
				Msg("invalid request data")

			return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		}

		// Getting alias
		alias, err := h.business.CreateURLAlias(ctx, item.Original)
		if err != nil {
			log.Error().
				Err(err).
				Str("correlation_id", item.ID).
				Str("original_url", item.Original).
				Msg("alias creation failed")

			return echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		// Add to response
		responseDataResult, _ := url.JoinPath(h.cfg.GetURLBase(), alias)
		responseData = append(responseData, jsonBatchResponseItem{
			ID:     item.ID,
			Result: responseDataResult,
		})
	}

	return c.JSON(http.StatusCreated, responseData)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.NewHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
}
