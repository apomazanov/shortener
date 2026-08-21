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

// BusinessService defines methods of business service used in handlers.
//
//go:generate mockgen -destination=mocks/mock_business.gen.go -package=mocks github.com/apomazanov/shortener/internal/handlers BusinessService
type BusinessService interface {
	GetOriginalURL(ctx context.Context, alias string) (original string, err error)
	GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error)
	CreateURLAlias(ctx context.Context, original string, userID string) (alias string, err error)
	CreateURLAliasBatch(ctx context.Context, originals []string, userID string) (written map[string]string, err error)
	DeleteUserURLs(ctx context.Context, userID string, aliases []string) error
}

// HealthService defines methods of health-check (ping) service used in handlers.
//
//go:generate mockgen -destination=mocks/mock_health.gen.go -package=mocks github.com/apomazanov/shortener/internal/handlers HealthService
type HealthService interface {
	Ping(ctx context.Context) error
}

// URLConfig defines methods of config used in handlers.
//
//go:generate mockgen -destination=mocks/mock_config.gen.go -package=mocks github.com/apomazanov/shortener/internal/handlers URLConfig
type URLConfig interface {
	GetURLBase() string
}

// JWT defines methods of JWT handling service used in handlers.
//
//go:generate mockgen -destination=mocks/mock_JWT.gen.go -package=mocks github.com/apomazanov/shortener/internal/handlers JWT
type JWT interface {
	CreateCookieWithUserID(userID string) (*http.Cookie, error)
}

// Handler contains data for handling application API endpoints.
type Handler struct {
	// business is a business service layer of application.
	business BusinessService
	// health is a health-check (ping) service.
	health HealthService
	// cfg is a configuration service.
	cfg URLConfig
	// log is a pointer to system logger.
	log *zerolog.Logger
	// jwt is a JWT handling helper.
	jwt JWT
}

// jsonShortenRequest defines data in shortening request (JSON format).
type jsonShortenRequest struct {
	// URL is a URL for shortening.
	URL string `json:"url" validate:"required,url"`
}

// jsonShortenResponse defines data in shortening response (JSON format).
type jsonShortenResponse struct {
	// Result is an alias value.
	Result string `json:"result"`
}

// jsonBatchRequestItem defines data in shortening request of batch of URLs.
type jsonBatchRequestItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id" validate:"required"`
	// Original is a URL for shortening.
	Original string `json:"original_url" validate:"required,url"`
}

// jsonBatchRequestItem defines data in shortening response of batch of URLs.
type jsonBatchResponseItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id"`
	// Result is an alias value.
	Result string `json:"short_url"`
}

// jsonGetUserURLsResponseItem defines data in response to 'GetUserURLs' request.
type jsonGetUserURLsResponseItem struct {
	// ShortURL is an alias value.
	ShortURL string `json:"short_url"`
	// OriginalURL is an original URL value.
	OriginalURL string `json:"original_url"`
}

// New creates a new handlers object.
func New(b BusinessService, c URLConfig, l *zerolog.Logger, h HealthService, j JWT) *Handler {
	return &Handler{
		business: b,
		health:   h,
		cfg:      c,
		log:      l,
		jwt:      j,
	}
}

// Get provides response to GET request of single alias value. It returns stored
// value of original URL, if found.
func (h *Handler) Get(c *echo.Context) error {

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Getting alias from request URL
	alias := c.Param("alias")

	if len(alias) != aliasSize {
		h.log.Info().
			Str("op", "handler.Get").
			Str("alias", alias).
			Msg("invalid alias")

		return echo.ErrBadRequest
	}

	// Find original URL
	original, err := h.business.GetOriginalURL(ctx, alias)

	if err == nil {
		c.Set("original", original)
		// Return original URL with redirection
		return c.Redirect(http.StatusTemporaryRedirect, original)
	}

	if errors.Is(err, domain.ErrFoundDeleted) {
		return c.NoContent(http.StatusGone)
	}

	if errors.Is(err, domain.ErrNotFound) {
		h.log.Info().
			Str("op", "handler.Get").
			Err(err).
			Msg("alias not found")

		return echo.ErrNotFound
	}

	h.log.Error().
		Str("op", "handler.Get").
		Err(err).
		Msg("something went wrong")

	return echo.ErrInternalServerError
}

// GetUserURLs provides response to GET request of all URLs which were stored by
// certain user.
func (h *Handler) GetUserURLs(c *echo.Context) error {

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Cookie

	cookieData := handleCookie(c, h.jwt)
	if cookieData.err != nil {
		h.log.Error().
			Str("op", "handler.GetUserURLs").
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

		h.log.Error().
			Str("op", "handler.GetUserURLs").
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

// Ping provides response to GET request of health-check.
func (h *Handler) Ping(c *echo.Context) error {

	ctx := c.Request().Context()

	if err := h.health.Ping(ctx); err != nil {

		h.log.Error().
			Str("op", "handler.Ping").
			Err(err).
			Msg("health check failed")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

// CreateText provides response to POST request of creating alias of certain
// URL (simple text format).
func (h *Handler) CreateText(c *echo.Context) error {

	// Cookie

	cookie := handleCookie(c, h.jwt)
	if cookie.err != nil {
		h.log.Error().
			Str("op", "handler.CreateText").
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
		h.log.Info().
			Str("op", "handler.CreateText").
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	// Casting body to string, validating
	requestData := jsonShortenRequest{URL: string(body)}
	if err := c.Validate(&requestData); err != nil {
		return echo.ErrBadRequest
	}

	responseStatus := http.StatusCreated

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			responseStatus = http.StatusConflict
		} else {
			h.log.Error().
				Str("op", "handler.CreateText").
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
		c.Set("user-id", userID)
	}

	c.Set("original", requestData.URL)

	// base URL already validated during config
	shortURL := h.cfg.GetURLBase() + "/" + alias
	return c.String(responseStatus, shortURL)
}

// CreateJSON provides response to POST request of creating alias of certain
// URL (JSON format).
func (h *Handler) CreateJSON(c *echo.Context) error {
	var requestData jsonShortenRequest
	var responseData jsonShortenResponse

	// Cookie

	cookie := handleCookie(c, h.jwt)
	if cookie.err != nil {
		h.log.Error().
			Str("op", "handler.CreateJSON").
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
		return echo.ErrBadRequest
	}

	// Validating
	if err := c.Validate(&requestData); err != nil {
		return echo.ErrBadRequest
	}

	responseStatus := http.StatusCreated

	// Getting alias
	alias, err := h.business.CreateURLAlias(ctx, requestData.URL, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			responseStatus = http.StatusConflict
		} else {
			h.log.Error().
				Str("op", "handler.CreateJSON").
				Err(err).
				Str("alias", alias).
				Msg("alias creation failed")

			return echo.ErrInternalServerError
		}
	}

	// Cookie created for new users (user-id missing in ctx or invalid)
	if !cookie.valid {
		c.SetCookie(cookie.newCookie)
		c.Set("user-id", userID)
	}

	c.Set("original", requestData.URL)

	// Sending back short URL
	responseData.Result = h.cfg.GetURLBase() + "/" + alias
	return c.JSON(responseStatus, responseData)
}

// CreateJSONBatch provides response to POST request of creating a batch of
// aliases to certain URLs.
func (h *Handler) CreateJSONBatch(c *echo.Context) error {

	// Cookie

	cookie := handleCookie(c, h.jwt)
	if cookie.err != nil {
		h.log.Error().
			Str("op", "handler.CreateJSONBatch").
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
		return echo.ErrBadRequest
	}

	batchSize := len(requestData)

	// Validating each item
	for _, item := range requestData {

		if err := c.Validate(&item); err != nil {
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
			h.log.Error().
				Str("op", "handler.CreateJSONBatch").
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
		c.Set("user-id", userID)
	}

	return c.JSON(responseStatus, responseData)
}

// DeleteUserURLs provides response to DELETE request of deleting all aliases
// created by certain user.
func (h *Handler) DeleteUserURLs(c *echo.Context) error {

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	// Cookie

	cookie := handleCookie(c, h.jwt)
	if cookie.err != nil {
		h.log.Error().
			Str("op", "handler.DeleteUserURLs").
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
		return echo.ErrBadRequest
	}

	// Validating
	if len(aliases) == 0 {
		return echo.ErrBadRequest
	}
	for _, alias := range aliases {
		if len(alias) != aliasSize {
			return echo.ErrBadRequest
		}
	}

	// Delete user's URLs
	err := h.business.DeleteUserURLs(ctx, cookie.userID, aliases)
	if err != nil {

		h.log.Error().
			Str("op", "handler.DeleteUserURLs").
			Err(err).
			Msg("something went wrong")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusAccepted)
}

// Reject is a handler of all unsupported requests.
func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.ErrBadRequest
}
