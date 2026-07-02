package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type cookieData struct {
	exists    bool
	valid     bool
	userID    string
	newCookie *http.Cookie
	newUserID string
	err       error
}

// if received cookie is OK, newCookie and newUserID are nil and ""
func handleCookie(c *echo.Context, jwt JWT) *cookieData {

	var result cookieData
	var ok bool

	result.exists, ok = extractCookieExistsFromCtx(c)
	if !ok {
		result.err = fmt.Errorf("Failed extracting cookie-exists from context")
		return &result
	}

	result.userID, ok = extractUserIDFromCtx(c)
	if !ok {
		result.err = fmt.Errorf("Failed extracting user-id from context")
		return &result
	}

	if _, err := uuid.Parse(result.userID); err == nil {
		result.valid = true
	}

	// New cookie needed in cases: 1) no cookie in req; 2) user-id in req invalid
	if !result.exists || !result.valid {
		result.newUserID = newUUIDString()
		var err error
		result.newCookie, err = jwt.CreateCookieWithUserID(result.newUserID)
		if err != nil {
			result.err = fmt.Errorf("Cookie creation failed: %w", err)
			return &result
		}
	}

	return &result
}

func extractUserIDFromCtx(c *echo.Context) (userID string, ok bool) {

	userIDAny := c.Get("user-id")

	if userIDAny == nil {
		// missing
		return "", true
	}

	userID, ok = userIDAny.(string)
	return userID, ok
}

func extractCookieExistsFromCtx(c *echo.Context) (exists bool, ok bool) {

	existsAny := c.Get("cookie-exists")

	if existsAny == nil {
		// missing
		return false, true
	}

	exists, ok = existsAny.(bool)
	return exists, ok
}

func newUUIDString() string {
	uuid := uuid.New()
	return uuid.String()
}
