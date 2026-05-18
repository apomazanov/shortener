package main

import (
	"log"
	"net/http"
	"os"

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
		// logger not initialized yet, using default log
		log.Fatalf("config error: %v\n", err)
	}

	l := logger.New() // TODO: level from config

	r, err := repository.NewFileRepo(cfg, l)
	if err != nil {
		// logged in repo layer
		return err
	}

	s := service.New(r, l)
	h := handlers.New(s, cfg, l)

	e := echo.New()
	e.Validator = validator.New()

	e.Use(middleware.Recover()) // always first to catch all following panics
	e.Use(middleware.RequestID()) // before logger, otherwise no requestIDs in logs
	e.Use(my_middleware.Zerologger(l)) // before other middlewares (full monitoring)
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		MinLength: 1024,
	}))
	e.Use(middleware.Decompress()) // strictly before body limit
	e.Use(middleware.BodyLimit(5_242_880)) // 5 Mb, avoiding OOM killer

	routes.Setup(e, h)

	// TODO: graceful shutdown
	err = e.Start(cfg.GetServerAddress())
	if err != nil {
		l.Fatal().
			Err(err).
			Msg("application terminated")
	}
	return err
}

/* -------------------------------------------------------------------------- */
func main() {
	if err := run(); err != nil && err != http.ErrServerClosed {
		os.Exit(1)
	}
}
