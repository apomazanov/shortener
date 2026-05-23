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

/* ------------------------------ Service mock ------------------------------ */
type serviceMock struct {
	original string
	alias    string
	err      error
}

func (s *serviceMock) GetOriginalURL(ctx context.Context, alias string) (original string, err error) {
	return s.original, s.err
}

func (s *serviceMock) CreateURLAlias(ctx context.Context, original string) (alias string, err error) {
	return s.alias, s.err

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
	s := &serviceMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	h := New(s, cfg, &l)
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
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusBadRequest, errHttp.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("google.com"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from service
		s.alias = ""
		s.err = errors.New("some error")
		v.err = nil

		err := h.CreateText(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusInternalServerError, errHttp.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("google.com"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from service
		s.alias = "short1"
		s.err = nil
		v.err = nil

		err := h.CreateText(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, cfg.baseURL+"/short1", resp.Body.String())
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_CreateJson(t *testing.T) {
	s := &serviceMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	h := New(s, cfg, &l)
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
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusBadRequest, errHttp.Code)
			assert.NotEmpty(t, errHttp.Message)
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
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusBadRequest, errHttp.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url": "https://google.com"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		s.alias = ""
		s.err = errors.New("some service error")
		v.err = nil

		err := h.CreateJson(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusInternalServerError, errHttp.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url": "https://google.com"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		s.alias = "short1"
		s.err = nil
		v.err = nil

		err := h.CreateJson(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.JSONEq(t, `{"result":"ba.se/short1"}`, resp.Body.String())
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_Get(t *testing.T) {
	s := &serviceMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	h := New(s, cfg, &l)
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
		s.original = "abracadabra1"
		s.err = nil

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
		s.original = ""
		s.err = domain.ErrNotFound

		err := h.Get(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusNotFound, errHttp.Code)
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
		s.original = ""
		s.err = errors.New("some error")

		err := h.Get(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusInternalServerError, errHttp.Code)
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
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusBadRequest, errHttp.Code)
		}
	})

}

/* -------------------------------------------------------------------------- */
func TestHandler_Reject(t *testing.T) {
	s := &serviceMock{}
	l := zerolog.New(os.Stdout)
	cfg := &cfgMock{baseURL: "ba.se", aliasSize: 6}
	h := New(s, cfg, &l)
	e := echo.New()

	t.Run("return BadRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)

		err := h.Reject(c)

		require.Error(t, err)
		http_err, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, http_err.Code)
	})
}
