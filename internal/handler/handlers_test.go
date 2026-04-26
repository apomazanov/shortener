package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* ----------------------------- Repository mock ---------------------------- */
type repoMock struct {
	value string
	ok bool
}

func newRepoMock() *repoMock {
	return &repoMock{}
}

func (r *repoMock) Add(long string) (string, bool) {
	return r.value, r.ok
}

func (r *repoMock) Get(short string) (string, bool) {
	return r.value, r.ok
}

/* -------------------------------------------------------------------------- */
func TestHandler_Register(t *testing.T) {
	r := newRepoMock()
	h := New(r)

	type expected struct {
		status int
		contentType string
		body string
	}

	tests := []struct {
		name string
		expected expected
		mocked repoMock
	} {
		{
			name: "success",
			expected: expected{
				status: http.StatusCreated,
				contentType: "text/plain",
				body: "http://localhost:8080/short1",
			},
			mocked: repoMock{
				value: "short1",
				ok: true,
			},
		},
		{
			name: "repo internal error",
			expected: expected{
				status: http.StatusServiceUnavailable,
				contentType: "text/plain",
				body: "Adding failed\n",
			},
			mocked: repoMock{
				value: "",
				ok: false,
			},
		},

		// TODO: io.ReadAll() error simulation
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// pre-test configuration
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			w := httptest.NewRecorder()
			r.value = test.mocked.value
			r.ok = test.mocked.ok

			// run handler
			h.RegisterURL(w, request)

			// getting results
			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			// checking
			require.NoError(t, err)
			assert.Equal(t, test.expected.status, res.StatusCode)
			assert.True(t, strings.HasPrefix(res.Header.Get("Content-Type"), test.expected.contentType))
			assert.Equal(t, test.expected.body, string(resBody))
		})
	}
}

/* -------------------------------------------------------------------------- */
func TestHandler_Extract(t *testing.T) {
	r := newRepoMock()
	h := New(r)

	type expected struct {
		status int
		contentType string
		location string
		body string
	}

	tests := []struct {
		name string
		expected expected
		mocked repoMock
	} {
		{
			name: "success",
			expected: expected{
				status: http.StatusTemporaryRedirect,
				contentType: "text/plain",
				location: "abracadabra1",
				body: "Redirecting to abracadabra1\n",
			},
			mocked: repoMock{
				value: "abracadabra1",
				ok: true,
			},
		},
		{
			name: "not found",
			expected: expected{
				status: http.StatusNotFound,
				contentType: "text/plain",
				body: "URL not found\n",
			},
			mocked: repoMock{
				value: "",
				ok: false,
			},
		},

		// TODO: io.ReadAll() error simulation
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// pre-test configuration
			request := httptest.NewRequest(http.MethodGet, "/", nil) // path is dummy, routing not checked
			w := httptest.NewRecorder()
			r.value = test.mocked.value
			r.ok = test.mocked.ok

			// run handler
			h.ExtractURL(w, request)

			// getting results
			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			// checking
			require.NoError(t, err)
			assert.Equal(t, test.expected.status, res.StatusCode)
			assert.True(t, strings.HasPrefix(res.Header.Get("Content-Type"), test.expected.contentType))
			assert.Equal(t, test.expected.body, string(resBody))
		})
	}
}

/* -------------------------------------------------------------------------- */
func TestHandler_Reject(t *testing.T) {
	r := newRepoMock()
	h := New(r)


	t.Run("return BadRequest", func(t *testing.T) {
			// pre-test configuration
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			w := httptest.NewRecorder()

			// run handler
			h.Reject(w, request)

			// getting results
			res := w.Result()

			// checking
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		})
}