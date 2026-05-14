package main

import (
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
		return err
	}

	l := logger.New() // TODO: level from config
	r := repository.NewInMemoryRepo()
	s := service.New(r, l)
	h := handlers.New(s, cfg, l)

	e := echo.New()
	e.Validator = validator.New()

	e.Use(middleware.RequestID())
	e.Use(my_middleware.Zerologger(l))
	e.Use(middleware.Recover())

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
