package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"

	"github.com/apomazanov/shortener/internal/config"
	"github.com/apomazanov/shortener/internal/handler"
	"github.com/apomazanov/shortener/internal/repository"
)

func run() error {
	// TODO: pass EnvVars as arguments ???
	cfg, err := config.New(os.Args[1:])
	if err != nil {
		return err
	}

	// TODO: service layer
	r := repository.NewInMemoryRepo()
	h := handler.New(r, cfg)

	e := echo.New()

	e.RouteNotFound("/*", h.Reject)
	e.GET("/:short", h.ExtractURL)
	e.POST("/", h.RegisterURL)

	// TODO: graceful shutdown
	return e.Start(cfg.GetServerAddress())
}

/* -------------------------------------------------------------------------- */
func main() {
	if err := run(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Application terminated: %v\r\n", err)
		os.Exit(1)
	}
}
