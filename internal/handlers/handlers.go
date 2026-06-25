package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

const aliasSize = 6

//go:generate mockgen -destination=mocks/mock_business.go -package=mocks github.com/apomazanov/shortener/internal/handlers BusinessService
type BusinessService interface {
	GetOriginalURL(ctx context.Context, alias string) (original string, err error)
	GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error)
	CreateURLAlias(ctx context.Context, original string, userID string) (alias string, err error)
	CreateURLAliasBatch(ctx context.Context, originals []string, userID string) (written map[string]string, err error)
}

//go:generate mockgen -destination=mocks/mock_health.go -package=mocks github.com/apomazanov/shortener/internal/handlers HealthService
type HealthService interface {
	Ping(ctx context.Context) error
}

//go:generate mockgen -destination=mocks/mock_config.go -package=mocks github.com/apomazanov/shortener/internal/handlers URLConfig
type URLConfig interface {
	GetURLBase() string
}

//go:generate mockgen -destination=mocks/mock_JWT.go -package=mocks github.com/apomazanov/shortener/internal/handlers JWT
type JWT interface {
	CreateCookieWithUserID(userID string) (*http.Cookie, error)
}

type Handler struct {
	business BusinessService
	health   HealthService
	cfg      URLConfig
	log      *zerolog.Logger
	JWT      JWT
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

type jsonGetUserURLsResponseItem struct {
	Alias    string `json:"short_url"`
	Original string `json:"original_url"`
}

func New(b BusinessService, c URLConfig, l *zerolog.Logger, h HealthService, j JWT) *Handler {
	return &Handler{
		business: b,
		health:   h,
		cfg:      c,
		log:      l,
		JWT:      j,
	}
}

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

		return echo.ErrBadRequest
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

		return echo.ErrNotFound
	}

	log.Error().
		Err(err).
		Msg("something went wrong")

	return echo.ErrInternalServerError
}

func (h *Handler) GetUserURLs(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.GetUserURLs").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// User ID
	ctxUserID, ok := extractUserIDFromCtx(c)
	if !ok {
		log.Error().
			Msg("Failed extracting user-id from context")

		return echo.ErrInternalServerError
	}

	if _, err := uuid.Parse(ctxUserID); err != nil {
		return echo.ErrUnauthorized
	}

	// Find user's URLs
	data, err := h.business.GetUserURLs(ctx, ctxUserID)
	if err != nil {

		if errors.Is(err, domain.ErrNotFound) {
			return c.NoContent(http.StatusNoContent)
		}

		log.Error().
			Err(err).
			Msg("something went wrong")

		return echo.ErrInternalServerError
	}

	responseData := make([]jsonGetUserURLsResponseItem, 0, len(data))

	for k, v := range data {
		responseData = append(responseData, jsonGetUserURLsResponseItem{
			Alias:    k,
			Original: v,
		})
	}

	return c.JSON(http.StatusOK, responseData)
}

func (h *Handler) Ping(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.Ping").Logger()

	ctx := c.Request().Context()

	if err := h.health.Ping(ctx); err != nil {

		log.Error().
			Err(err).
			Msg("health check failed")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) CreateText(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.CreateText").Logger()

	// User ID
	ctxUserID, ok := extractUserIDFromCtx(c)
	if !ok {
		log.Error().
			Msg("Failed extracting user-id from context")

		return echo.ErrInternalServerError
	}

	userID := ctxUserID
	if _, err := uuid.Parse(userID); err != nil {
		userID = newUUIDString()
	}

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body, expecting long URL for shortening
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	// Casting body to string, validating
	requestData := jsonShortenRequest{URL: string(body)}
	if err := c.Validate(&requestData); err != nil {
		log.Info().
			Err(err).
			Any("request_data", requestData).
			Msg("invalid request data")

		return echo.ErrBadRequest
	}

	responseStatus := http.StatusCreated

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			responseStatus = http.StatusConflict
		} else {
			log.Error().
				Err(err).
				Str("alias", alias).
				Msg("alias creation failed")

			return echo.ErrInternalServerError
		}
	}

	// Sending back short URL

	// Cookie created for new users (user-id missing in ctx or invalid)
	if ctxUserID == "" || ctxUserID != userID {
		newCookie, err := h.JWT.CreateCookieWithUserID(userID)
		if err != nil {
			log.Error().
				Err(err).
				Msg("cookie creation failed")
			return echo.ErrInternalServerError
		}

		c.SetCookie(newCookie)
	}

	// base URL already validated during config
	shortUrl := h.cfg.GetURLBase() + "/" + alias
	return c.String(responseStatus, shortUrl)
}

func (h *Handler) CreateJson(c *echo.Context) error {
	var requestData jsonShortenRequest
	var responseData jsonShortenResponse

	log := h.log.With().Str("op", "handler.CreateJson").Logger()

	// User ID
	ctxUserID, ok := extractUserIDFromCtx(c)
	if !ok {
		log.Error().
			Msg("Failed extracting user-id from context")

		return echo.ErrInternalServerError
	}

	userID := ctxUserID
	if _, err := uuid.Parse(userID); err != nil {
		userID = newUUIDString()
	}

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body
	if err := c.Bind(&requestData); err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	// Validating
	if err := c.Validate(&requestData); err != nil {
		log.Info().
			Err(err).
			Any("request_data", requestData).
			Msg("invalid request data")

		return echo.ErrBadRequest
	}

	responseStatus := http.StatusCreated

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			responseStatus = http.StatusConflict
		} else {
			log.Error().
				Err(err).
				Str("alias", alias).
				Msg("alias creation failed")

			return echo.ErrInternalServerError
		}
	}

	// Cookie created for new users (user-id missing in ctx or invalid)
	if ctxUserID == "" || ctxUserID != userID {
		newCookie, err := h.JWT.CreateCookieWithUserID(userID)
		if err != nil {
			log.Error().
				Err(err).
				Msg("cookie creation failed")
			return echo.ErrInternalServerError
		}

		c.SetCookie(newCookie)
	}

	// Sending back short URL
	responseData.Result, _ = url.JoinPath(h.cfg.GetURLBase(), alias)
	return c.JSON(responseStatus, responseData)
}

func (h *Handler) CreateJsonBatch(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.CreateJsonBatch").Logger()

	// User ID
	ctxUserID, ok := extractUserIDFromCtx(c)
	if !ok {
		log.Error().
			Msg("Failed extracting user-id from context")

		return echo.ErrInternalServerError
	}

	userID := ctxUserID
	if _, err := uuid.Parse(userID); err != nil {
		userID = newUUIDString()
	}

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Reading body
	var requestData []jsonBatchRequestItem

	if err := c.Bind(&requestData); err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	batchSize := len(requestData)

	// Validating each item
	for _, item := range requestData {

		if err := c.Validate(&item); err != nil {
			log.Info().
				Err(err).
				Any("request_item", item).
				Msg("invalid request data")

			return echo.ErrBadRequest
		}
	}

	// Preparing list of originals

	originals := make([]string, batchSize)
	for i, item := range requestData {
		originals[i] = item.Original
	}

	responseStatus := http.StatusCreated

	// Getting aliases

	written, err := h.business.CreateURLAliasBatch(ctx, originals, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			responseStatus = http.StatusConflict
		} else {
			log.Error().
				Err(err).
				Msg("alias creation failed")

			return echo.ErrInternalServerError
		}
	}

	// Filling response

	// Same size as request. Even if there are URLs duplicates, correlation-id is major priority
	responseData := make([]jsonBatchResponseItem, batchSize)

	for i := range batchSize {
		original := requestData[i].Original
		alias := written[original]
		responseDataResult, _ := url.JoinPath(h.cfg.GetURLBase(), alias)
		responseData[i] = jsonBatchResponseItem{
			ID:     requestData[i].ID,
			Result: responseDataResult,
		}
	}

	// Cookie created for new users (user-id missing in ctx or invalid)
	if ctxUserID == "" || ctxUserID != userID {
		newCookie, err := h.JWT.CreateCookieWithUserID(userID)
		if err != nil {
			log.Error().
				Err(err).
				Msg("cookie creation failed")
			return echo.ErrInternalServerError
		}

		c.SetCookie(newCookie)
	}

	return c.JSON(responseStatus, responseData)
}

func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.ErrBadRequest
}

func extractUserIDFromCtx(c *echo.Context) (userID string, ok bool) {

	userIDAny := c.Get("user-id")

	if userIDAny == nil {
		// missing
		return "", true
	}

	userID, ok = userIDAny.(string)
	return userID, ok
}

func newUUIDString() string {
	uuid := uuid.New()
	return uuid.String()
}
