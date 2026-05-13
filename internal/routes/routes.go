package routes

import (
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/labstack/echo/v5"
)

/* -------------------------------------------------------------------------- */
func Setup(e *echo.Echo, userHandler *handlers.Handler) {

	e.GET("/:short", userHandler.Get)
	e.POST("/", userHandler.Create)
	e.RouteNotFound("/*", userHandler.Reject)
}
