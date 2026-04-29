package main

import (
	"flag"

	"github.com/labstack/echo/v5"

	"github.com/apomazanov/shortener/internal/config"
	"github.com/apomazanov/shortener/internal/handler"
	"github.com/apomazanov/shortener/internal/repository"
)

/* -------------------------------------------------------------------------- */
func main() {
	cfg := &config.Config{}

	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Base URL for aliases")
	flag.StringVar(&cfg.ServerPort, "a", ":8080", "HTTP-server address:port")
	flag.Parse()

	// TODO: service layer
	r := repository.NewInMemoryRepo()
	h := handler.New(r, cfg)

	e := echo.New()

	e.RouteNotFound("/*", h.Reject)
	e.GET("/:short", h.ExtractURL)
	e.POST("/", h.RegisterURL)

	// TODO: graceful shutdown
	err := e.Start(cfg.GetServerAddress())
	if err != nil {
		panic(err)
	}
}
