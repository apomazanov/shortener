package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* ------------------------------ Business mock ----------------------------- */
type businessMock struct {
	original string
	alias    string
	err      error
}

func (b *businessMock) GetOriginalURL(ctx context.Context, alias string) (original string, err error) {
	return b.original, b.err
}

func (b *businessMock) CreateURLAlias(ctx context.Context, original string) (alias string, err error) {
	return b.alias, b.err
}

/* ------------------------------- Health mock ------------------------------ */
type healthMock struct {
	err error
}

func (h *healthMock) Ping(ctx context.Context) error {
	return h.err
}

/* ------------------------------- Config mock ------------------------------ */
type cfgMock struct {
	baseURL   string
	aliasSize int
}

func (c *cfgMock) GetURLBase() string {
	return c.baseURL
}

func (c *cfgMock) GetAliasSize() int {
	return c.aliasSize
}

/* ----------------------------- Validator mock ----------------------------- */
type valMock struct {
	err error
}

func (v *valMock) Validate(i any) error {
	return v.err
}

/* -------------------------------------------------------------------------- */
func TestHandler_CreateText(t *testing.T) {
	b := &businessMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	health := &healthMock{}
	h := New(b, cfg, &l, health)
	v := valMock{}
	e := echo.New()
	e.Validator = &v

	t.Run("validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		v.err = errors.New("validation error")

		err := h.CreateText(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusBadRequest, errHTTP.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("google.com"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from service
		b.alias = ""
		b.err = errors.New("some error")
		v.err = nil

		err := h.CreateText(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusInternalServerError, errHTTP.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("google.com"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from service
		b.alias = "short1"
		b.err = nil
		v.err = nil

		err := h.CreateText(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, cfg.baseURL+"/short1", resp.Body.String())
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_CreateJson(t *testing.T) {
	b := &businessMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	health := &healthMock{}
	h := New(b, cfg, &l, health)
	v := valMock{}
	e := echo.New()
	e.Validator = &v

	t.Run("malformed json body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		v.err = nil

		err := h.CreateJson(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusBadRequest, errHTTP.Code)
			assert.NotEmpty(t, errHTTP.Message)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url": ""}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		v.err = errors.New("validation error")

		err := h.CreateJson(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusBadRequest, errHTTP.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url": "https://google.com"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		b.alias = ""
		b.err = errors.New("some service error")
		v.err = nil

		err := h.CreateJson(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusInternalServerError, errHTTP.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url": "https://google.com"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		b.alias = "short1"
		b.err = nil
		v.err = nil

		err := h.CreateJson(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.JSONEq(t, `{"result":"ba.se/short1"}`, resp.Body.String())
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_Get(t *testing.T) {
	b := &businessMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	health := &healthMock{}
	h := New(b, cfg, &l, health)
	e := echo.New()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:alias")
		c.SetPathValues(echo.PathValues{
			{Name: "alias", Value: "abcdef"},
		})
		// mocking response from service
		b.original = "abracadabra1"
		b.err = nil

		err := h.Get(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, resp.Code)
		assert.Equal(t, "abracadabra1", resp.Header().Get("Location"))
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:alias")
		c.SetPathValues(echo.PathValues{
			{Name: "alias", Value: "abcdef"},
		})
		// mocking response from service
		b.original = ""
		b.err = domain.ErrNotFound

		err := h.Get(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusNotFound, errHTTP.Code)
		}
	})

	t.Run("other error from service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:alias")
		c.SetPathValues(echo.PathValues{
			{Name: "alias", Value: "abcdef"},
		})
		// mocking response from service
		b.original = ""
		b.err = errors.New("some error")

		err := h.Get(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusInternalServerError, errHTTP.Code)
		}
	})

	t.Run("invalid alias", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:alias")
		c.SetPathValues(echo.PathValues{
			{Name: "alias", Value: "abcde"}, // size < cfg.aliasSize
		})

		err := h.Get(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusBadRequest, errHTTP.Code)
		}
	})

}

/* -------------------------------------------------------------------------- */
func TestHandler_Ping(t *testing.T) {
	b := &businessMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	health := &healthMock{}
	h := New(b, cfg, &l, health)
	e := echo.New()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		health.err = nil

		err := h.Ping(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("health check failed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		health.err = errors.New("ping error")

		err := h.Ping(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusInternalServerError, errHTTP.Code)
		}
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_Reject(t *testing.T) {
	b := &businessMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	health := &healthMock{}
	h := New(b, cfg, &l, health)
	e := echo.New()

	t.Run("return BadRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)

		err := h.Reject(c)

		require.Error(t, err)
		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			assert.Equal(t, http.StatusBadRequest, errHTTP.Code)
		}
	})
}
