package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/internal/handlers/mocks"
	"github.com/apomazanov/shortener/internal/validator"
)

type testContext struct {
	e   *echo.Echo
	ctx *echo.Context
	res *httptest.ResponseRecorder
}

type mocksContainer struct {
	service *mocks.MockBusinessService
	config  *mocks.MockURLConfig
	health  *mocks.MockHealthService
	jwt     *mocks.MockJWT
}

type ctxOptions struct {
	method      string
	alias       string
	userID      any
	body        string
	contentType string
}

func createMocksAndHandler(t *testing.T) (m *mocksContainer, h *Handler) {
	t.Helper()

	c := gomock.NewController(t)
	m = &mocksContainer{
		service: mocks.NewMockBusinessService(c),
		config:  mocks.NewMockURLConfig(c),
		health:  mocks.NewMockHealthService(c),
		jwt:     mocks.NewMockJWT(c),
	}
	l := zerolog.Nop()
	h = New(m.service, m.config, &l, m.health, m.jwt)

	return m, h
}

func createCtx(t *testing.T, opts *ctxOptions) *testContext {
	t.Helper()

	e := echo.New()
	e.Validator = validator.New()

	req := httptest.NewRequest(opts.method, "/"+opts.alias, strings.NewReader(opts.body))
	if opts.contentType != "" {
		req.Header.Set(echo.HeaderContentType, opts.contentType)
	}

	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	if opts.alias != "" {
		ctx.SetPath("/:alias")
		ctx.SetPathValues(echo.PathValues{
			{Name: "alias", Value: opts.alias},
		})
	}

	if opts.userID != nil {
		ctx.Set("user-id", opts.userID)
		ctx.Set("cookie-exists", true)
	}

	return &testContext{e: e, ctx: ctx, res: res}
}

func TestHandler_Get(t *testing.T) {

	const (
		validAlias   = "abcdef"
		invalidAlias = "abc"
	)

	tests := []struct {
		name             string
		alias            string
		mocksSetup       func(m *mocksContainer)
		expectedErr      error
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:  "success, redirection",
			alias: validAlias,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("http://example.com/original", nil).
					Times(1)
			},
			expectedErr:      nil,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "http://example.com/original",
		},
		{
			name:             "invalid alias size",
			alias:            invalidAlias,
			mocksSetup:       nil,
			expectedErr:      echo.ErrBadRequest,
			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
		},
		{
			name:  "not found",
			alias: validAlias,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("", domain.ErrNotFound).
					Times(1)
			},
			expectedErr:      echo.ErrNotFound,
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		{
			name:  "internal error",
			alias: validAlias,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("", errors.New("undefined error")).
					Times(1)
			},
			expectedErr:      echo.ErrInternalServerError,
			expectedStatus:   http.StatusInternalServerError,
			expectedLocation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// tcx := createCtxGet(t, tt.alias, nil)
			tcx := createCtx(t, &ctxOptions{
				method: http.MethodGet,
				alias:  tt.alias,
			})
			m, h := createMocksAndHandler(t)
			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.Get(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)
			assert.Equal(t, tt.expectedLocation, tcx.res.Header().Get("Location"))
		})
	}
}

func TestHandler_Ping(t *testing.T) {

	tests := []struct {
		name           string
		mocksSetup     func(m *mocksContainer)
		expectedErr    error
		expectedStatus int
	}{
		{
			name: "success",
			mocksSetup: func(m *mocksContainer) {
				m.health.EXPECT().
					Ping(gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedErr:    nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "fail",
			mocksSetup: func(m *mocksContainer) {
				m.health.EXPECT().
					Ping(gomock.Any()).
					Return(errors.New("health error")).
					Times(1)
			},
			expectedErr:    echo.ErrInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// tcx := createCtxGet(t, "dummyPath", nil)
			tcx := createCtx(t, &ctxOptions{
				method: http.MethodGet,
			})
			m, h := createMocksAndHandler(t)
			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.Ping(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)
		})
	}
}

func TestHandler_GetUserURLs(t *testing.T) {
	const (
		validUserID   = "123e4567-e89b-42d3-a456-426614174000"
		invalidUserID = "super puper root user"
		baseURL       = "http://shortener"
	)

	tests := []struct {
		name             string
		userID           any
		mocksSetup       func(m *mocksContainer)
		expectedErr      error
		expectedStatus   int
		expectedResponse []jsonGetUserURLsResponseItem
	}{
		{
			name:   "success",
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(map[string]string{
						"alias1": "original1",
						"alias2": "original2",
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(2)
			},
			expectedErr:    nil,
			expectedStatus: http.StatusOK,
			expectedResponse: []jsonGetUserURLsResponseItem{
				{
					ShortURL:    baseURL + "/alias1",
					OriginalURL: "original1",
				},
				{
					ShortURL:    baseURL + "/alias2",
					OriginalURL: "original2",
				},
			},
		},
		{
			name:   "cookie invalid",
			userID: invalidUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), gomock.Any()).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Times(1)
			},
			expectedErr:    echo.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "user-id missing",
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), gomock.Any()).
					Times(0)
				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)
			},
			expectedErr:    nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "user-id extraction error",
			userID: 123,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:    echo.ErrInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "not found",
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(nil, domain.ErrNotFound).
					Times(1)
			},
			expectedErr:    nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "unknown service error",
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(nil, errors.New("unknown error")).
					Times(1)
			},
			expectedErr:    echo.ErrInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createCtx(t, &ctxOptions{
				method: http.MethodGet,
				userID: tt.userID,
			})
			m, h := createMocksAndHandler(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.GetUserURLs(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)

			if tt.expectedStatus == http.StatusOK {
				var received []jsonGetUserURLsResponseItem

				require.NoError(t, json.Unmarshal(tcx.res.Body.Bytes(), &received))
				assert.ElementsMatch(t, tt.expectedResponse, received)
			}

			if tt.expectedStatus == http.StatusNoContent {
				assert.Empty(t, tcx.res.Body.String())
			}
		})
	}
}

func TestHandler_CreateText(t *testing.T) {
	const (
		validUserID = "123e4567-e89b-42d3-a456-426614174000"
		validURL    = "http://example.com/original"
		baseURL     = "http://shortener"
		alias       = "abcdef"
	)

	tests := []struct {
		name              string
		body              string
		userID            any
		mocksSetup        func(m *mocksContainer)
		expectedErr       error
		expectedStatus    int
		expectedBody      string
		expectedSetCookie bool
	}{
		{
			name:   "success existing user",
			body:   validURL,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(alias, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedBody:      baseURL + "/" + alias,
			expectedSetCookie: false,
		},
		{
			name:   "success new user",
			body:   validURL,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, gomock.Any()).
					Return(alias, nil).
					Times(1)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedBody:      baseURL + "/" + alias,
			expectedSetCookie: true,
		},
		{
			name:   "invalid url",
			body:   "not-url",
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      "",
			expectedSetCookie: false,
		},
		{
			name:   "user-id has invalid type",
			body:   validURL,
			userID: 123,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      "",
			expectedSetCookie: false,
		},
		{
			name:   "duplicate original url",
			body:   validURL,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(alias, domain.ErrOriginalURLDuplicate).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusConflict,
			expectedBody:      baseURL + "/" + alias,
			expectedSetCookie: false,
		},
		{
			name:   "service internal error",
			body:   validURL,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return("", errors.New("create alias error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      "",
			expectedSetCookie: false,
		},
		{
			name:   "cookie creation error",
			body:   validURL,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, gomock.Any()).
					Return(alias, nil).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(nil, errors.New("cookie error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      "",
			expectedSetCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createCtx(t, &ctxOptions{
				method: http.MethodPost,
				userID: tt.userID,
				body:   tt.body,
			})

			m, h := createMocksAndHandler(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.CreateText(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)

			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, tcx.res.Body.String())
			}

			if tt.expectedSetCookie {
				assert.NotEmpty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			} else {
				assert.Empty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			}
		})
	}
}

func TestHandler_CreateJson(t *testing.T) {
	const (
		validUserID = "123e4567-e89b-42d3-a456-426614174000"
		validURL    = "http://example.com/original"
		baseURL     = "http://shortener"
		alias       = "abcdef"
	)

	tests := []struct {
		name              string
		body              string
		userID            any
		mocksSetup        func(m *mocksContainer)
		expectedErr       error
		expectedStatus    int
		expectedResponse  jsonShortenResponse
		expectedSetCookie bool
	}{
		{
			name:   "success existing user",
			body:   `{"url":"` + validURL + `"}`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(alias, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  jsonShortenResponse{Result: baseURL + "/" + alias},
			expectedSetCookie: false,
		},
		{
			name:   "success new user",
			body:   `{"url":"` + validURL + `"}`,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, gomock.Any()).
					Return(alias, nil).
					Times(1)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  jsonShortenResponse{Result: baseURL + "/" + alias},
			expectedSetCookie: true,
		},
		{
			name:   "invalid json",
			body:   `{"url":`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  jsonShortenResponse{},
			expectedSetCookie: false,
		},
		{
			name:   "invalid url",
			body:   `{"url":"not-url"}`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  jsonShortenResponse{},
			expectedSetCookie: false,
		},
		{
			name:   "user-id extraction error",
			body:   `{"url":"` + validURL + `"}`,
			userID: 123,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  jsonShortenResponse{},
			expectedSetCookie: false,
		},
		{
			name:   "invalid user-id creates new user and cookie",
			body:   `{"url":"` + validURL + `"}`,
			userID: "not-a-uuid",
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, gomock.Any()).
					Return(alias, nil).
					Times(1)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  jsonShortenResponse{Result: baseURL + "/" + alias},
			expectedSetCookie: true,
		},
		{
			name:   "duplicate original url",
			body:   `{"url":"` + validURL + `"}`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(alias, domain.ErrOriginalURLDuplicate).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusConflict,
			expectedResponse:  jsonShortenResponse{Result: baseURL + "/" + alias},
			expectedSetCookie: false,
		},
		{
			name:   "service internal error",
			body:   `{"url":"` + validURL + `"}`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return("", errors.New("create alias error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  jsonShortenResponse{},
			expectedSetCookie: false,
		},
		{
			name:   "cookie creation error",
			body:   `{"url":"` + validURL + `"}`,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, gomock.Any()).
					Return(alias, nil).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(nil, errors.New("cookie error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  jsonShortenResponse{},
			expectedSetCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createCtx(t, &ctxOptions{
				method:      http.MethodPost,
				userID:      tt.userID,
				body:        tt.body,
				contentType: echo.MIMEApplicationJSON,
			})

			m, h := createMocksAndHandler(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.CreateJSON(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)

			if tt.expectedResponse.Result != "" {
				var received jsonShortenResponse

				require.NoError(t, json.Unmarshal(tcx.res.Body.Bytes(), &received))
				assert.Equal(t, tt.expectedResponse, received)
			}

			if tt.expectedSetCookie {
				assert.NotEmpty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			} else {
				assert.Empty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			}
		})
	}
}

func TestHandler_CreateJsonBatch(t *testing.T) {
	const (
		validUserID = "123e4567-e89b-42d3-a456-426614174000"
		baseURL     = "http://shortener"

		validURL1 = "http://example.com/original-1"
		validURL2 = "http://example.com/original-2"

		alias1 = "abcde1"
		alias2 = "abcde2"
	)

	validBody := `[
		{
			"correlation_id":"id-1",
			"original_url":"` + validURL1 + `"
		},
		{
			"correlation_id":"id-2",
			"original_url":"` + validURL2 + `"
		}
	]`

	expectedSuccessResponse := []jsonBatchResponseItem{
		{
			ID:     "id-1",
			Result: baseURL + "/" + alias1,
		},
		{
			ID:     "id-2",
			Result: baseURL + "/" + alias2,
		},
	}

	tests := []struct {
		name              string
		body              string
		userID            any
		mocksSetup        func(m *mocksContainer)
		expectedErr       error
		expectedStatus    int
		expectedResponse  []jsonBatchResponseItem
		expectedSetCookie bool
	}{
		{
			name:   "success existing user",
			body:   validBody,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						validUserID,
					).
					Return(map[string]string{
						validURL1: alias1,
						validURL2: alias2,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(2)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  expectedSuccessResponse,
			expectedSetCookie: false,
		},
		{
			name:   "success new user",
			body:   validBody,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						gomock.Any(),
					).
					Return(map[string]string{
						validURL1: alias1,
						validURL2: alias2,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(2)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  expectedSuccessResponse,
			expectedSetCookie: true,
		},
		{
			name:   "invalid json",
			body:   `[{"correlation_id":`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
		{
			name: "invalid item missing correlation id",
			body: `[
				{
					"original_url":"` + validURL1 + `"
				}
			]`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
		{
			name: "invalid item url",
			body: `[
				{
					"correlation_id":"id-1",
					"original_url":"not-url"
				}
			]`,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
		{
			name:   "user-id extraction error",
			body:   validBody,
			userID: 123,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
		{
			name:   "invalid user-id creates new user and cookie",
			body:   validBody,
			userID: "not-a-uuid",
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						gomock.Any(),
					).
					Return(map[string]string{
						validURL1: alias1,
						validURL2: alias2,
					}, nil).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(2)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(&http.Cookie{
						Name:  "user-id",
						Value: validUserID,
					}, nil).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusCreated,
			expectedResponse:  expectedSuccessResponse,
			expectedSetCookie: true,
		},
		{
			name:   "duplicate original url",
			body:   validBody,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						validUserID,
					).
					Return(map[string]string{
						validURL1: alias1,
						validURL2: alias2,
					}, domain.ErrOriginalURLDuplicate).
					Times(1)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(2)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusConflict,
			expectedResponse:  expectedSuccessResponse,
			expectedSetCookie: false,
		},
		{
			name:   "service internal error",
			body:   validBody,
			userID: validUserID,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						validUserID,
					).
					Return(nil, errors.New("create aliases error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
		{
			name:   "cookie creation error",
			body:   validBody,
			userID: nil,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					CreateURLAliasBatch(
						gomock.Any(),
						[]string{validURL1, validURL2},
						gomock.Any(),
					).
					Return(map[string]string{
						validURL1: alias1,
						validURL2: alias2,
					}, nil).
					Times(0)

				m.config.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Return(nil, errors.New("cookie error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  nil,
			expectedSetCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createCtx(t, &ctxOptions{
				method:      http.MethodPost,
				userID:      tt.userID,
				body:        tt.body,
				contentType: echo.MIMEApplicationJSON,
			})

			m, h := createMocksAndHandler(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.CreateJSONBatch(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)

			if len(tt.expectedResponse) > 0 {
				var received []jsonBatchResponseItem

				require.NoError(t, json.Unmarshal(tcx.res.Body.Bytes(), &received))
				assert.Equal(t, tt.expectedResponse, received)
			}

			if tt.expectedSetCookie {
				assert.NotEmpty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			} else {
				assert.Empty(t, tcx.res.Header().Values(echo.HeaderSetCookie))
			}
		})
	}
}

func TestHandler_Reject(t *testing.T) {
	tcx := createCtx(t, &ctxOptions{
		method: http.MethodGet,
		alias:  "invalid/path",
	})

	_, h := createMocksAndHandler(t)

	err := h.Reject(tcx.ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, echo.ErrBadRequest)
	require.Equal(t, http.StatusBadRequest, echo.StatusCode(err))

	tcx.e.HTTPErrorHandler(tcx.ctx, err)

	assert.Equal(t, http.StatusBadRequest, tcx.res.Code)
}

func TestHandler_DeleteUserURLsByAlias(t *testing.T) {
	const (
		validUserID   = "123e4567-e89b-42d3-a456-426614174000"
		invalidUserID = "not-a-valid-uuid"
		baseURL       = "http://shortener"
		alias1        = "alias1"
		alias2        = "alias2"
	)

	validBody := `["` + alias1 + `", "` + alias2 + `"]`

	tests := []struct {
		name              string
		userID            any
		body              string
		mocksSetup        func(m *mocksContainer)
		expectedErr       error
		expectedStatus    int
		expectedSetCookie bool
	}{
		{
			name:   "success",
			userID: validUserID,
			body:   validBody,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), validUserID, []string{alias1, alias2}).
					Return(nil).
					Times(1)
			},
			expectedErr:       nil,
			expectedStatus:    http.StatusAccepted,
			expectedSetCookie: false,
		},
		{
			name:   "unauthorized without cookie",
			userID: nil,
			body:   validBody,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Times(1)
			},
			expectedErr:       echo.ErrUnauthorized,
			expectedStatus:    http.StatusUnauthorized,
			expectedSetCookie: false,
		},
		{
			name:   "user-id invalid uuid, unauthorized",
			userID: invalidUserID,
			body:   validBody,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				m.jwt.EXPECT().
					CreateCookieWithUserID(gomock.Any()).
					Times(1)
			},
			expectedErr:       echo.ErrUnauthorized,
			expectedStatus:    http.StatusUnauthorized,
			expectedSetCookie: false,
		},
		{
			name:   "empty request data",
			userID: validUserID,
			body:   "",
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedSetCookie: false,
		},
		{
			name:   "invalid alias size",
			userID: validUserID,
			body:   `["123", "456"]`,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedErr:       echo.ErrBadRequest,
			expectedStatus:    http.StatusBadRequest,
			expectedSetCookie: false,
		},
		{
			name:   "service internal error",
			userID: validUserID,
			body:   validBody,
			mocksSetup: func(m *mocksContainer) {
				m.service.EXPECT().
					DeleteUserURLs(gomock.Any(), validUserID, []string{alias1, alias2}).
					Return(errors.New("internal service error")).
					Times(1)
			},
			expectedErr:       echo.ErrInternalServerError,
			expectedStatus:    http.StatusInternalServerError,
			expectedSetCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createCtx(t, &ctxOptions{
				method:      http.MethodPost,
				userID:      tt.userID,
				body:        tt.body,
				contentType: echo.MIMEApplicationJSON,
			})

			m, h := createMocksAndHandler(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(m)
			}

			err := h.DeleteUserURLsByAlias(tcx.ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, tt.expectedStatus, echo.StatusCode(err))
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				tcx.e.HTTPErrorHandler(tcx.ctx, err)
			}

			assert.Equal(t, tt.expectedStatus, tcx.res.Code)
		})
	}
}
