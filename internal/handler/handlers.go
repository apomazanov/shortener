package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
)

type URLRepository interface {
	Add(long string) (string, bool)
	Get(short string) (string, bool)
}

type Handler struct {
	repo URLRepository
}

/* -------------------------------------------------------------------------- */
func New(repo URLRepository) *Handler {
	return &Handler{repo: repo}
}

/* -------------------------------------------------------------------------- */
func (h *Handler) ExtractURL(c *echo.Context) error {
	short := c.Param("short")

	long, ok := h.repo.Get(short)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "URL not found")
	}

	return c.Redirect(http.StatusTemporaryRedirect, long)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) RegisterURL(c *echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	body_str := string(body)
	if body_str == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "No URL passed")
	}

	short, ok := h.repo.Add(body_str)
	if !ok {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Adding failed")
	}

	short = fmt.Sprintf("http://localhost:8080/%s", short)
	return c.String(http.StatusCreated, short)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Reject(c *echo.Context) error {
	return echo.NewHTTPError(http.StatusBadRequest, "")
}
