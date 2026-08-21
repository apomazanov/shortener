package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// cookieData contains data for handling cookie object.
type cookieData struct {
	// exists flag shows if cookie exists in a request.
	exists bool
	// valid shows if cookie is valid.
	valid bool
	// userID is user-id value stored in token.
	userID string
	// newCookie is a pointer to new cookie object (with newUserID value).
	newCookie *http.Cookie
	// newUserID is a new user-id value.
	newUserID string
	// err is an error value that might be set during cookie handling.
	err error
}

// handleCookie handles cookie object and fills data to cookieData.
// If received cookie is OK, newCookie and newUserID are nil and ""
func handleCookie(c *echo.Context, jwt JWT) *cookieData {

	var result cookieData
	var ok bool

	result.exists, ok = extractCookieExistsFromCtx(c)
	if !ok {
		result.err = fmt.Errorf("failed extracting cookie-exists from context")
		return &result
	}

	result.userID, ok = extractUserIDFromCtx(c)
	if !ok {
		result.err = fmt.Errorf("failed extracting user-id from context")
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
			result.err = fmt.Errorf("cookie creation failed: %w", err)
			return &result
		}
	}

	return &result
}

// extractUserIDFromCtx helps to extract user-id value from context.
func extractUserIDFromCtx(c *echo.Context) (userID string, ok bool) {

	userIDAny := c.Get("user-id")

	if userIDAny == nil {
		// missing
		return "", true
	}

	userID, ok = userIDAny.(string)
	return userID, ok
}

// extractCookieExistsFromCtx helps to extract value of parameter 'cookie-exists' from context.
func extractCookieExistsFromCtx(c *echo.Context) (exists bool, ok bool) {

	existsAny := c.Get("cookie-exists")

	if existsAny == nil {
		// missing
		return false, true
	}

	exists, ok = existsAny.(bool)
	return exists, ok
}

// newUUIDString generates a new user-id value.
func newUUIDString() string {
	uuid := uuid.New()
	return uuid.String()
}
