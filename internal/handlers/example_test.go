package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/apomazanov/shortener/internal/handlers/mocks"
	"github.com/apomazanov/shortener/internal/validator"
)

const (
	exampleUserID = "123e4567-e89b-42d3-a456-426614174000"
	exampleURL    = "https://practicum.yandex.ru/"
	exampleAlias  = "abc123"
	exampleBase   = "http://shortener.local"
)

type noopT struct{}

func (noopT) Errorf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
func (noopT) Fatalf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
func (noopT) Helper()                           {}
func (noopT) Cleanup(_ func())                  {}
func (noopT) Logf(_ string, _ ...any)           {}
func (noopT) Name() string                      { return "example" }
func (noopT) Setenv(_, _ string)                {}

func setupHandler() (*handlers.Handler, *mocks.MockBusinessService, *mocks.MockURLConfig, *mocks.MockHealthService, *mocks.MockJWT) {
	ctrl := gomock.NewController(&noopT{})

	svc := mocks.NewMockBusinessService(ctrl)
	cfg := mocks.NewMockURLConfig(ctrl)
	health := mocks.NewMockHealthService(ctrl)
	jwt := mocks.NewMockJWT(ctrl)

	log := zerolog.Nop()
	h := handlers.New(svc, cfg, &log, health, jwt)

	return h, svc, cfg, health, jwt
}

func makeContext(method, path, body string, contentType string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = validator.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)
	return e, ctx, res
}

func ExampleHandler_Get() {
	h, svc, _, _, _ := setupHandler()

	svc.EXPECT().
		GetOriginalURL(gomock.Any(), exampleAlias).
		Return(exampleURL, nil).
		Times(1)

	_, ctx, res := makeContext(http.MethodGet, "/"+exampleAlias, "", "")
	ctx.SetPath("/:alias")
	ctx.SetPathValues(echo.PathValues{
		{Name: "alias", Value: exampleAlias},
	})

	_ = h.Get(ctx)

	fmt.Println("status:", res.Code)
	fmt.Println("location:", res.Header().Get("Location"))
	// Output:
	// status: 307
	// location: https://practicum.yandex.ru/
}

func ExampleHandler_Get_notFound() {
	h, svc, _, _, _ := setupHandler()

	svc.EXPECT().
		GetOriginalURL(gomock.Any(), exampleAlias).
		Return("", domain.ErrNotFound).
		Times(1)

	_, ctx, res := makeContext(http.MethodGet, "/"+exampleAlias, "", "")
	ctx.SetPath("/:alias")
	ctx.SetPathValues(echo.PathValues{
		{Name: "alias", Value: exampleAlias},
	})

	err := h.Get(ctx)
	if err != nil {
		ctx.Echo().HTTPErrorHandler(ctx, err)
	}

	fmt.Println("status:", res.Code)
	// Output:
	// status: 404
}

func ExampleHandler_Get_deleted() {
	h, svc, _, _, _ := setupHandler()

	svc.EXPECT().
		GetOriginalURL(gomock.Any(), exampleAlias).
		Return("", domain.ErrFoundDeleted).
		Times(1)

	_, ctx, res := makeContext(http.MethodGet, "/"+exampleAlias, "", "")
	ctx.SetPath("/:alias")
	ctx.SetPathValues(echo.PathValues{
		{Name: "alias", Value: exampleAlias},
	})

	_ = h.Get(ctx)

	fmt.Println("status:", res.Code)
	// Output:
	// status: 410
}

func ExampleHandler_CreateText() {
	h, svc, cfg, _, _ := setupHandler()

	svc.EXPECT().
		CreateURLAlias(gomock.Any(), exampleURL, exampleUserID).
		Return(exampleAlias, nil).
		Times(1)

	cfg.EXPECT().
		GetURLBase().
		Return(exampleBase).
		Times(1)

	_, ctx, res := makeContext(http.MethodPost, "/", exampleURL, "")
	ctx.Set("user-id", exampleUserID)
	ctx.Set("cookie-exists", true)

	_ = h.CreateText(ctx)

	fmt.Println("status:", res.Code)
	fmt.Println("body:", res.Body.String())
	// Output:
	// status: 201
	// body: http://shortener.local/abc123
}

func ExampleHandler_CreateJSON() {
	h, svc, cfg, _, _ := setupHandler()

	svc.EXPECT().
		CreateURLAlias(gomock.Any(), exampleURL, exampleUserID).
		Return(exampleAlias, nil).
		Times(1)

	cfg.EXPECT().
		GetURLBase().
		Return(exampleBase).
		Times(1)

	body := `{"url":"` + exampleURL + `"}`
	_, ctx, res := makeContext(http.MethodPost, "/api/shorten", body, echo.MIMEApplicationJSON)
	ctx.Set("user-id", exampleUserID)
	ctx.Set("cookie-exists", true)

	_ = h.CreateJSON(ctx)

	var resp struct {
		Result string `json:"result"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &resp)

	fmt.Println("status:", res.Code)
	fmt.Println("result:", resp.Result)
	// Output:
	// status: 201
	// result: http://shortener.local/abc123
}

func ExampleHandler_CreateJSONBatch() {
	h, svc, cfg, _, _ := setupHandler()

	originals := []string{
		"https://example.com/page-1",
		"https://example.com/page-2",
	}
	aliases := map[string]string{
		originals[0]: "aaa111",
		originals[1]: "bbb222",
	}

	svc.EXPECT().
		CreateURLAliasBatch(gomock.Any(), originals, exampleUserID).
		Return(aliases, nil).
		Times(1)

	cfg.EXPECT().
		GetURLBase().
		Return(exampleBase).
		AnyTimes()

	body := `[
		{"correlation_id":"id-1","original_url":"https://example.com/page-1"},
		{"correlation_id":"id-2","original_url":"https://example.com/page-2"}
	]`
	_, ctx, res := makeContext(http.MethodPost, "/api/shorten/batch", body, echo.MIMEApplicationJSON)
	ctx.Set("user-id", exampleUserID)
	ctx.Set("cookie-exists", true)

	_ = h.CreateJSONBatch(ctx)

	var resp []struct {
		ID     string `json:"correlation_id"`
		Result string `json:"short_url"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &resp)

	fmt.Println("status:", res.Code)
	for _, item := range resp {
		fmt.Printf("  %s -> %s\n", item.ID, item.Result)
	}
	// Output:
	// status: 201
	//   id-1 -> http://shortener.local/aaa111
	//   id-2 -> http://shortener.local/bbb222
}

func ExampleHandler_GetUserURLs() {
	h, svc, cfg, _, _ := setupHandler()

	svc.EXPECT().
		GetUserURLs(gomock.Any(), exampleUserID).
		Return(map[string]string{
			"aaa111": "https://example.com/page-1",
			"bbb222": "https://example.com/page-2",
		}, nil).
		Times(1)

	cfg.EXPECT().
		GetURLBase().
		Return(exampleBase).
		AnyTimes()

	_, ctx, res := makeContext(http.MethodGet, "/api/user/urls", "", "")
	ctx.Set("user-id", exampleUserID)
	ctx.Set("cookie-exists", true)

	_ = h.GetUserURLs(ctx)

	fmt.Println("status:", res.Code)

	var resp []struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &resp)

	for _, item := range resp {
		fmt.Printf("  %s -> %s\n", item.ShortURL, item.OriginalURL)
	}
	// Output:
	// status: 200
	//   http://shortener.local/aaa111 -> https://example.com/page-1
	//   http://shortener.local/bbb222 -> https://example.com/page-2
}

func ExampleHandler_Ping() {
	h, _, _, health, _ := setupHandler()

	health.EXPECT().
		Ping(gomock.Any()).
		Return(nil).
		Times(1)

	_, ctx, res := makeContext(http.MethodGet, "/ping", "", "")

	_ = h.Ping(ctx)

	fmt.Println("status:", res.Code)
	// Output:
	// status: 200
}

func ExampleHandler_DeleteUserURLs() {
	h, svc, _, _, _ := setupHandler()

	aliases := []string{"aaa111", "bbb222"}

	svc.EXPECT().
		DeleteUserURLs(gomock.Any(), exampleUserID, aliases).
		Return(nil).
		Times(1)

	body := `["aaa111","bbb222"]`
	_, ctx, res := makeContext(http.MethodDelete, "/api/user/urls", body, echo.MIMEApplicationJSON)
	ctx.Set("user-id", exampleUserID)
	ctx.Set("cookie-exists", true)

	_ = h.DeleteUserURLs(ctx)

	fmt.Println("status:", res.Code)
	// Output:
	// status: 202
}
