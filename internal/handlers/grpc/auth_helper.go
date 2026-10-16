package grpc_handlers

import (
	"context"
	"fmt"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/google/uuid"
)

// authData contains data for authorization process.
type authData struct {
	// exists flag shows if auth data exists in a request.
	exists bool
	// valid shows if auth data is valid.
	valid bool
	// userID is user-id value stored in token.
	userID string
	// newUserID is a new user-id value.
	newUserID string
	// newToken is a new token value for newUserID.
	newToken string
	// err is an error value that might be set during cookie handling.
	err error
}

// handleAuthData handles context and fills data to authData.
// If received auth data is OK, newCookie and newUserID are nil and ""
func handleAuthData(ctx context.Context, jwt domain.JWT) *authData {

	var result authData
	var ok bool

	result.exists, ok = extractTokenExistsFromCtx(ctx)
	if !ok {
		result.err = fmt.Errorf("failed extracting token-exists from context")
		return &result
	}

	result.userID, ok = extractUserIDFromCtx(ctx)
	if !ok {
		result.err = fmt.Errorf("failed extracting user-id from context")
		return &result
	}

	if _, err := uuid.Parse(result.userID); err == nil {
		result.valid = true
	}

	// New user-id needed in cases: 1) no auth data in req; 2) user-id in req invalid
	if !result.exists || !result.valid {
		result.newUserID = newUUIDString()
		var err error
		result.newToken, err = jwt.CreateTokenWithUserID(result.newUserID)
		if err != nil {
			result.err = fmt.Errorf("token creation failed: %w", err)
		}
	}

	return &result
}

// extractUserIDFromCtx helps to extract user-id value from context.
func extractUserIDFromCtx(ctx context.Context) (userID string, ok bool) {

	userIDAny := ctx.Value("user-id")

	if userIDAny == nil {
		// missing
		return "", true
	}

	userID, ok = userIDAny.(string)
	return userID, ok
}

// extractTokenExistsFromCtx helps to extract value of parameter 'token-exists' from context.
func extractTokenExistsFromCtx(ctx context.Context) (exists bool, ok bool) {

	existsAny := ctx.Value("token-exists")

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
