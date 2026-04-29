package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
)

type URLRepository interface {
	Add(long string) (short string, ok bool)
	Get(short string) (long string, ok bool)
}

type URLConfig interface {
	GetURLBase() string
}

type Handler struct {
	repo URLRepository
	cfg  URLConfig
}

/* -------------------------------------------------------------------------- */
func New(repo URLRepository, cfg URLConfig) *Handler {
	return &Handler{
		repo: repo,
		cfg:  cfg,
	}
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
		return echo.NewHTTPError(http.StatusInternalServerError, "Adding failed")
	}

	short = fmt.Sprintf("%s/%s", h.cfg.GetURLBase(), short)
	return c.String(http.StatusCreated, short)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Reject(c *echo.Context) error {
	return echo.NewHTTPError(http.StatusBadRequest, "")
}
