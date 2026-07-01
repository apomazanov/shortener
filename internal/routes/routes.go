package routes

import (
	"github.com/apomazanov/shortener/internal/handlers"
	"github.com/labstack/echo/v5"
)

/* -------------------------------------------------------------------------- */
func Setup(e *echo.Echo, h *handlers.Handler, authMiddleware echo.MiddlewareFunc) {

	e.GET("/:alias", h.Get)
	e.GET("/ping", h.Ping)
	e.GET("/api/user/urls", h.GetUserURLs, authMiddleware)
	e.POST("/", h.CreateText, authMiddleware)
	e.POST("/api/shorten", h.CreateJson, authMiddleware)
	e.POST("/api/shorten/batch", h.CreateJsonBatch, authMiddleware)
	e.DELETE("/api/user/urls", h.DeleteUserURLsByAlias, authMiddleware)
	e.RouteNotFound("/*", h.Reject)
}
