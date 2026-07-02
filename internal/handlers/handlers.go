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

//go:generate mockgen -destination=mocks/mock_business.go -package=mocks github.com/apomazanov/shortener/internal/handlers BusinessService
type BusinessService interface {
	GetOriginalURL(ctx context.Context, alias string) (original string, err error)
	GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error)
	CreateURLAlias(ctx context.Context, original string, userID string) (alias string, err error)
	CreateURLAliasBatch(ctx context.Context, originals []string, userID string) (written map[string]string, err error)
	DeleteUserURLs(ctx context.Context, userID string, aliases []string) error
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
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
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

	if errors.Is(err, domain.ErrFoundDeleted) {
		return c.NoContent(http.StatusGone)
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

	// Cookie

	cookieData := handleCookie(c, h.JWT)
	if cookieData.err != nil {
		log.Error().
			Err(cookieData.err).
			Msg("Failed cookie handling")

		return echo.ErrInternalServerError
	}

	if !cookieData.exists {
		c.SetCookie(cookieData.newCookie)
		return c.NoContent(http.StatusNoContent)
	}

	// Invalid user-id leads to auth failure
	if !cookieData.valid {
		return echo.ErrUnauthorized
	}

	// Find user's URLs
	data, err := h.business.GetUserURLs(ctx, cookieData.userID)
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
			ShortURL:    h.cfg.GetURLBase() + "/" + k,
			OriginalURL: v,
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

	// Cookie

	cookie := handleCookie(c, h.JWT)
	if cookie.err != nil {
		log.Error().
			Err(cookie.err).
			Msg("Failed cookie handling")

		return echo.ErrInternalServerError
	}

	userID := cookie.userID
	if !cookie.valid {
		userID = cookie.newUserID
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
	if !cookie.valid {
		c.SetCookie(cookie.newCookie)
	}

	// base URL already validated during config
	shortUrl := h.cfg.GetURLBase() + "/" + alias
	return c.String(responseStatus, shortUrl)
}

func (h *Handler) CreateJson(c *echo.Context) error {
	var requestData jsonShortenRequest
	var responseData jsonShortenResponse

	log := h.log.With().Str("op", "handler.CreateJson").Logger()

	// Cookie

	cookie := handleCookie(c, h.JWT)
	if cookie.err != nil {
		log.Error().
			Err(cookie.err).
			Msg("Failed cookie handling")

		return echo.ErrInternalServerError
	}

	userID := cookie.userID
	if !cookie.valid {
		userID = cookie.newUserID
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
	if !cookie.valid {
		c.SetCookie(cookie.newCookie)
	}

	// Sending back short URL
	responseData.Result, _ = url.JoinPath(h.cfg.GetURLBase(), alias)
	return c.JSON(responseStatus, responseData)
}

func (h *Handler) CreateJsonBatch(c *echo.Context) error {

	log := h.log.With().Str("op", "handler.CreateJsonBatch").Logger()

	// Cookie

	cookie := handleCookie(c, h.JWT)
	if cookie.err != nil {
		log.Error().
			Err(cookie.err).
			Msg("Failed cookie handling")

		return echo.ErrInternalServerError
	}

	userID := cookie.userID
	if !cookie.valid {
		userID = cookie.newUserID
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
	if !cookie.valid {
		c.SetCookie(cookie.newCookie)
	}

	return c.JSON(responseStatus, responseData)
}

func (h *Handler) DeleteUserURLsByAlias(c *echo.Context) error {
	log := h.log.With().Str("op", "handler.DeleteUserURLs").Logger()

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Cookie

	cookie := handleCookie(c, h.JWT)
	if cookie.err != nil {
		log.Error().
			Err(cookie.err).
			Msg("Failed cookie handling")

		return echo.ErrInternalServerError
	}

	// Missing cookie or invalid user-id lead to auth failure
	if !cookie.exists || !cookie.valid {
		return echo.ErrUnauthorized
	}

	var aliases []string

	// Reading body
	if err := c.Bind(&aliases); err != nil {
		log.Info().
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	// Validating
	if len(aliases) == 0 {
		log.Info().
			Msg("empty request data")

		return echo.ErrBadRequest
	}
	for _, alias := range aliases {
		if len(alias) != aliasSize {
			log.Info().
				Str("alias", alias).
				Msg("invalid alias")

			return echo.ErrBadRequest
		}
	}

	// Delete user's URLs
	err := h.business.DeleteUserURLs(ctx, cookie.userID, aliases)
	if err != nil {

		log.Error().
			Err(err).
			Msg("something went wrong")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusAccepted)
}

func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.ErrBadRequest
}
