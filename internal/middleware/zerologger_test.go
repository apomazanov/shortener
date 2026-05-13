package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* -------------------------------------------------------------------------- */
func TestGetResponseData(t *testing.T) {
	e := echo.New()

	t.Run("status from response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Response().WriteHeader(http.StatusAccepted)

		status, _ := getResponseData(c, nil)
		assert.Equal(t, http.StatusAccepted, status)
	})

	t.Run("status from HTTPError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := echo.NewHTTPError(http.StatusBadRequest, "bad request")

		status, _ := getResponseData(c, err)
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("status from generic error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := errors.New("internal error")

		status, _ := getResponseData(c, err)
		assert.Equal(t, http.StatusInternalServerError, status)
	})

	t.Run("default to OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		status, _ := getResponseData(c, nil)
		assert.Equal(t, http.StatusOK, status)
	})
}

/* -------------------------------------------------------------------------- */
func TestZerologger(t *testing.T) {
	e := echo.New()
	buf := new(bytes.Buffer)
	l := zerolog.New(buf)
	middleware := Zerologger(&l)

	t.Run("log success", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c *echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), `"method":"GET"`)
		assert.Contains(t, buf.String(), `"uri":"/test"`)
		assert.Contains(t, buf.String(), `"status":200`)
	})

	t.Run("log error", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest(http.MethodPost, "/fail", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c *echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		})

		err := handler(c)
		require.Error(t, err)
		assert.Contains(t, buf.String(), `"method":"POST"`)
		assert.Contains(t, buf.String(), `"uri":"/fail"`)
		assert.Contains(t, buf.String(), `"status":404`)
	})
}
