package middleware

// CtxKey used as key values in context.
type CtxKey string

// UserIDKey is a key for storing user-id value in context
const UserIDKey CtxKey = "user-id"

// CookieExistsKey is a key for storing a fact that cookie exists in request
const CookieExistsKey CtxKey = "cookie-exists"

// TokenExistsKey is a key for storing a fact that token exists in request
const TokenExistsKey CtxKey = "token-exists"
