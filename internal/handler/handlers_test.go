package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* ----------------------------- Repository mock ---------------------------- */
type repoMock struct {
	value string
	ok    bool
}

func (r *repoMock) Add(long string) (string, bool) {
	return r.value, r.ok
}

func (r *repoMock) Get(short string) (string, bool) {
	return r.value, r.ok
}

/* ------------------------------- Config mock ------------------------------ */
type cfgMock struct {
	baseURL string
}

func (c *cfgMock) GetURLBase() string {
	return c.baseURL
}

/* -------------------------------------------------------------------------- */
func TestHandler_Register(t *testing.T) {
	r := &repoMock{}
	cfg := &cfgMock{}
	h := New(r, cfg)
	e := echo.New()

	// TODO: io.ReadAll() error simulation

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from repo
		r.value = ""
		r.ok = false

		err := h.RegisterURL(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusBadRequest, errHttp.Code)
			assert.Equal(t, "No URL passed", errHttp.Message)
		}
	})

	t.Run("repo internal error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abracadabra"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from repo
		r.value = ""
		r.ok = false

		err := h.RegisterURL(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusInternalServerError, errHttp.Code)
			assert.Equal(t, "Adding failed", errHttp.Message)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abracadabra"))
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		// mocking response from repo
		r.value = "short1"
		r.ok = true

		err := h.RegisterURL(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, cfg.baseURL+"/short1", resp.Body.String())
	})
}

/* -------------------------------------------------------------------------- */
func TestHandler_Extract(t *testing.T) {
	r := &repoMock{}
	cfg := &cfgMock{}
	h := New(r, cfg)
	e := echo.New()

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:short")
		c.SetPathValues(echo.PathValues{
			{Name: "short", Value: "short_URL"}, // value doesn't matter
		})
		// mocking response from repo
		r.value = "abracadabra1"
		r.ok = true

		err := h.ExtractURL(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, resp.Code)
		assert.Equal(t, "abracadabra1", resp.Header().Get("Location"))
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
		resp := httptest.NewRecorder()
		c := e.NewContext(req, resp)
		c.SetPath("/:short")
		c.SetPathValues(echo.PathValues{
			{Name: "short", Value: "short_URL"}, // value doesn't matter
		})
		// mocking response from repo
		r.value = ""
		r.ok = false

		err := h.ExtractURL(c)

		require.Error(t, err)
		var errHttp *echo.HTTPError
		if errors.As(err, &errHttp) {
			assert.Equal(t, http.StatusNotFound, errHttp.Code)
			assert.Equal(t, "URL not found", errHttp.Message)
		}
	})

}

/* -------------------------------------------------------------------------- */
func TestHandler_Reject(t *testing.T) {
	r := &repoMock{}
	cfg := &cfgMock{}
	h := New(r, cfg)
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
