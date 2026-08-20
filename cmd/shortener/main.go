package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo-contrib/v5/pprof"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/apomazanov/shortener/internal/audit"
	"github.com/apomazanov/shortener/internal/config"
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/apomazanov/shortener/internal/repository"
	"github.com/apomazanov/shortener/internal/service"
	"github.com/apomazanov/shortener/internal/validator"
	"github.com/apomazanov/shortener/pkg/logger"

	my_jwt "github.com/apomazanov/shortener/internal/jwt"
	my_middleware "github.com/apomazanov/shortener/internal/middleware"
)

type Repo interface {
	service.Repo
	Close() error
	Ping(ctx context.Context) error
}

func run(log *zerolog.Logger) error {

	cfg, err := config.New(os.Args[1:])
	if err != nil {
		return fmt.Errorf("run: config error: %w", err)
	}

	// Repository

	var r Repo

	if cfg.GetDatabaseDSN() != "" {
		log.Info().Msg("run: initializing postgres")
		r, err = repository.NewPostgresStorage(cfg)
	} else if cfg.GetStorageFile() != "" {
		log.Info().Msg("run: initializing local storage")
		r, err = repository.NewLocalStorage(cfg, log)
	} else {
		log.Info().Msg("run: initializing memory storage")
		r = repository.NewMemStorage()
	}

	if err != nil {
		return fmt.Errorf("run: repo create error: %w", err)
	}

	defer func() {
		if err := r.Close(); err != nil {
			log.Error().
				Err(err).
				Msg("run: repo close error")
		}
	}()

	// JWT

	jwtData := my_jwt.Data{
		SigningKey: []byte(cfg.GetJWTSecret()),
		CookieName: "auth_token",
		TokenTTL:   24 * time.Hour,
		CookieTTL:  72 * time.Hour,
	}

	// Service and handlers

	s := service.New(r, &service.AsyncDeleterConfig{
		BatchSize: 10,
		Timeout:   time.Second * 3,
	}, log)

	h := handlers.New(s, cfg, log, r, &jwtData)

	e := echo.New()
	e.Validator = validator.New()

	// Audit

	auditCtx, auditCancel := context.WithCancel(context.Background())
	defer auditCancel()

	var wgAudit sync.WaitGroup

	auditor := audit.NewDispatcher(log)
	wgAudit.Go(func() {
		auditor.Run(auditCtx)
	})

	if cfg.GetAuditFile() != "" {
		auditToFile := audit.NewAuditToFile(cfg.GetAuditFile(), log)
		auditor.Register(auditToFile)
		wgAudit.Go(func() {
			auditToFile.Run(auditCtx)
		})
	}

	if cfg.GetAuditURL() != "" {
		auditToURL := audit.NewAuditToURL(cfg.GetAuditURL(), log)
		auditor.Register(auditToURL)
		wgAudit.Go(func() {
			auditToURL.Run(auditCtx)
		})
	}

	// Middleware

	e.Use(middleware.Recover())          // always first to catch all following panics
	e.Use(middleware.RequestID())        // before logger, otherwise no requestIDs in logs
	e.Use(my_middleware.Zerologger(log)) // before other middlewares (full monitoring)
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		MinLength: 1024,
	}))
	e.Use(middleware.Decompress())         // strictly before body limit
	e.Use(middleware.BodyLimit(5_242_880)) // 5 Mb, avoiding OOM killer

	authMiddleware := my_middleware.Authenticator(&jwtData, log)
	auditMiddleware := my_middleware.AuditRecorder(auditor.InputChannel(), log)

	// Routes

	e.GET("/:alias", h.Get, auditMiddleware)
	e.GET("/ping", h.Ping)
	e.GET("/api/user/urls", h.GetUserURLs, authMiddleware)
	e.POST("/", h.CreateText, authMiddleware, auditMiddleware)
	e.POST("/api/shorten", h.CreateJSON, authMiddleware, auditMiddleware)
	e.POST("/api/shorten/batch", h.CreateJSONBatch, authMiddleware)
	e.DELETE("/api/user/urls", h.DeleteUserURLs, authMiddleware)
	e.RouteNotFound("/*", h.Reject)

	pprof.Register(e)

	// Graceful shutdown

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         cfg.GetServerAddress(),
		GracefulTimeout: 10 * time.Second,
	}

	err = sc.Start(ctx, e) // blocking, HTTP-server running

	// HTTP-server gracefully shut down, ready to disable audit services
	time.AfterFunc(10*time.Second, func() {
		// emergency shutdown in case of stuck
		auditCancel()
	})
	auditor.Stop()
	wgAudit.Wait()

	return err
}

func main() {
	log := logger.New()

	if err := run(log); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().
			Err(err).
			Msg("application terminated")

		os.Exit(1)
	}
}
