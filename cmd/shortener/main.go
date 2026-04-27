package main

import (
	"github.com/labstack/echo/v5"

	"github.com/apomazanov/shortener/internal/handler"
	"github.com/apomazanov/shortener/internal/repository"
)

/* -------------------------------------------------------------------------- */
func main() {
	r := repository.NewInMemoryRepo()
	h := handler.New(r)

	e := echo.New()

	e.RouteNotFound("/*", h.Reject)
	e.GET("/:short", h.ExtractURL)
	e.POST("/", h.RegisterURL)

	err := e.Start("127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
}
