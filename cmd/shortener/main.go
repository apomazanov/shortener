package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/apomazanov/shortener/internal/config"
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/apomazanov/shortener/internal/repository"
	"github.com/apomazanov/shortener/internal/routes"
	"github.com/apomazanov/shortener/internal/service"
	"github.com/apomazanov/shortener/internal/validator"
	"github.com/apomazanov/shortener/pkg/logger"

	my_middleware "github.com/apomazanov/shortener/internal/middleware"
)

/* -------------------------------------------------------------------------- */
func run() error {
	// TODO: pass EnvVars as arguments ???
	cfg, err := config.New(os.Args[1:])
	if err != nil {
		// logger not initialized yet, using fmt
		return fmt.Errorf("config error: %w", err)
	}

	l := logger.New() // TODO: level from config

	r, err := repository.NewFileRepo(cfg, l)
	if err != nil {
		// logged in repo layer
		return fmt.Errorf("repo error: %w", err)
	}

	s := service.New(r, l)
	h := handlers.New(s, cfg, l)

	e := echo.New()
	e.Validator = validator.New()

	// Middleware

	e.Use(middleware.Recover())        // always first to catch all following panics
	e.Use(middleware.RequestID())      // before logger, otherwise no requestIDs in logs
	e.Use(my_middleware.Zerologger(l)) // before other middlewares (full monitoring)
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
	if err := run(); err != nil && err != http.ErrServerClosed {
		os.Exit(1)
	}
}
