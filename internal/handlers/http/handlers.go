package http_handlers

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

const aliasSize = 6

// Handler contains data for handling application API endpoints.
type Handler struct {
	// business is a business service layer of application.
	business domain.BusinessService
	// health is a health-check (ping) service.
	health domain.HealthService
	// cfg is a configuration service.
	cfg domain.URLConfig
	// log is a pointer to system logger.
	log *zerolog.Logger
	// jwt is a JWT handling helper.
	jwt domain.JWT
}

// New creates a new handlers object.
func New(b domain.BusinessService, c domain.URLConfig, l *zerolog.Logger, h domain.HealthService, j domain.JWT) *Handler {
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
			Str("op", "http_handler.Get").
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
			Str("op", "http_handler.Get").
			Err(err).
			Msg("alias not found")

		return echo.ErrNotFound
	}

	h.log.Error().
		Str("op", "http_handler.Get").
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
			Str("op", "http_handler.GetUserURLs").
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
			Str("op", "http_handler.GetUserURLs").
			Err(err).
			Msg("something went wrong")

		return echo.ErrInternalServerError
	}

	responseData := make([]domain.JsonGetUserURLsResponseItem, 0, len(data))

	for k, v := range data {
		responseData = append(responseData, domain.JsonGetUserURLsResponseItem{
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
			Str("op", "http_handler.Ping").
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
			Str("op", "http_handler.CreateText").
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
			Str("op", "http_handler.CreateText").
			Err(err).
			Msg("failed to read request body")

		return echo.ErrBadRequest
	}

	// Casting body to string, validating
	requestData := domain.JsonShortenRequest{URL: string(body)}
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
				Str("op", "http_handler.CreateText").
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
	var requestData domain.JsonShortenRequest
	var responseData domain.JsonShortenResponse

	// Cookie

	cookie := handleCookie(c, h.jwt)
	if cookie.err != nil {
		h.log.Error().
			Str("op", "http_handler.CreateJSON").
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
				Str("op", "http_handler.CreateJSON").
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
			Str("op", "http_handler.CreateJSONBatch").
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
	var requestData []domain.JsonBatchRequestItem

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
				Str("op", "http_handler.CreateJSONBatch").
				Err(err).
				Msg("alias creation failed")

			return echo.ErrInternalServerError
		}
	}

	// Filling response

	// Same size as request. Even if there are URLs duplicates, correlation-id is major priority
	responseData := make([]domain.JsonBatchResponseItem, batchSize)

	for i := range batchSize {
		original := requestData[i].Original
		alias := written[original]
		responseDataResult, _ := url.JoinPath(h.cfg.GetURLBase(), alias)
		responseData[i] = domain.JsonBatchResponseItem{
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
			Str("op", "http_handler.DeleteUserURLs").
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
			Str("op", "http_handler.DeleteUserURLs").
			Err(err).
			Msg("something went wrong")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusAccepted)
}

// GetStats provides response to statistics request. Only trusted IP-addresses will be responded.
func (h *Handler) GetStats(c *echo.Context) error {

	// Raw context for deeper layers, evading 'echo' dependency
	ctx := c.Request().Context()

	stats, err := h.business.GetStats(ctx)
	if err != nil {
		h.log.Error().
			Str("op", "http_handler.GetStats").
			Err(err).
			Msg("stats request failed")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, stats)
}

// Reject is a handler of all unsupported requests.
func (h *Handler) Reject(c *echo.Context) error {

	h.log.Warn().
		Str("op", "http_handler.Reject").
		Str("remote_ip", c.RealIP()).
		Msg("invalid path in request")

	return echo.ErrBadRequest
}
