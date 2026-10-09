package middleware

import "github.com/labstack/echo/v5"

type nextHandlerMock struct {
	handler func(c *echo.Context) error
}

func (m *nextHandlerMock) Handler(c *echo.Context) error {
	return m.handler(c)
}
