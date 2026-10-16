package domain

//go:generate mockgen -source=service.go -destination=mocks/mock_service.gen.go -package=mocks

import (
	"context"
	"net/http"
)

// BusinessService defines methods of business service used in handlers.
type BusinessService interface {
	GetOriginalURL(ctx context.Context, alias string) (original string, err error)
	GetUserURLs(ctx context.Context, userID string) (data map[string]string, err error)
	CreateURLAlias(ctx context.Context, original string, userID string) (alias string, err error)
	CreateURLAliasBatch(ctx context.Context, originals []string, userID string) (written map[string]string, err error)
	DeleteUserURLs(ctx context.Context, userID string, aliases []string) error
	GetStats(ctx context.Context) (Stats, error)
}

// HealthService defines methods of health-check (ping) service used in handlers.
type HealthService interface {
	Ping(ctx context.Context) error
}

// URLConfig defines methods of config used in handlers.
type URLConfig interface {
	GetURLBase() string
}

// JWT defines methods of JWT handling service used in handlers.
type JWT interface {
	CreateCookieWithUserID(userID string) (*http.Cookie, error)
	CreateTokenWithUserID(userID string) (string, error)
}
