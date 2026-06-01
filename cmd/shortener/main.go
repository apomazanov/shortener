package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/apomazanov/shortener/internal/config"
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/apomazanov/shortener/internal/repository"
	"github.com/apomazanov/shortener/internal/routes"
	"github.com/apomazanov/shortener/internal/service"
	"github.com/apomazanov/shortener/internal/validator"
	"github.com/apomazanov/shortener/pkg/logger"

	my_middleware "github.com/apomazanov/shortener/internal/middleware"
)

type Repo interface {
	service.Repo
	Close() error
	Ping(ctx context.Context) error
}

/* -------------------------------------------------------------------------- */
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
		r = repository.NewMemStorage(log)
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

	// Service and handlers

	s := service.New(r)
	h := handlers.New(s, cfg, log, r)

	e := echo.New()
	e.Validator = validator.New()

	// Middleware

	e.Use(middleware.Recover())          // always first to catch all following panics
	e.Use(middleware.RequestID())        // before logger, otherwise no requestIDs in logs
	e.Use(my_middleware.Zerologger(log)) // before other middlewares (full monitoring)
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		MinLength: 1024,
	}))
	e.Use(middleware.Decompress())         // strictly before body limit
	e.Use(middleware.BodyLimit(5_242_880)) // 5 Mb, avoiding OOM killer

	// Routes

	routes.Setup(e, h)

	// Graceful shutdown

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         cfg.GetServerAddress(),
		GracefulTimeout: 10 * time.Second,
	}

	return sc.Start(ctx, e)
}

/* -------------------------------------------------------------------------- */
func main() {
	log := logger.New()

	if err := run(log); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().
			Err(err).
			Msg("application terminated")

		os.Exit(1)
	}
}
