package routes

import (
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/labstack/echo/v5"
)

/* -------------------------------------------------------------------------- */
func Setup(e *echo.Echo, userHandler *handlers.Handler) {

	e.GET("/:alias", userHandler.Get)
	e.GET("/ping", userHandler.Ping)
	e.POST("/", userHandler.CreateText)
	e.POST("/api/shorten", userHandler.CreateJson)
	e.RouteNotFound("/*", userHandler.Reject)
}
