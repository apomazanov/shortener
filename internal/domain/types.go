package domain

// AuditEvent defines an event for audit service.
type AuditEvent struct {
	// Timestamp is an event creation timestamp (unix format).
	Timestamp int64 `json:"ts"`
	// Action is a type of event.
	Action string `json:"action"`
	// UserID is a creator of event (request author).
	UserID string `json:"user_id,omitempty"`
	// URL is an original URL that was handled by request.
	URL string `json:"url"`
}

// Stats defines statistics data.
type Stats struct {
	// Aliases is a quantity of aliases stored in repo.
	Aliases int `json:"urls"`
	// Users is a quantity of users that have undeleted aliases in repo.
	Users int `json:"users"`
}

// JsonShortenRequest defines data in shortening request (JSON format).
type JsonShortenRequest struct {
	// URL is a URL for shortening.
	URL string `json:"url" validate:"required,url"`
}

// JsonShortenResponse defines data in shortening response (JSON format).
type JsonShortenResponse struct {
	// Result is an alias value.
	Result string `json:"result"`
}

// JsonBatchRequestItem defines data in shortening request of batch of URLs.
type JsonBatchRequestItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id" validate:"required"`
	// Original is a URL for shortening.
	Original string `json:"original_url" validate:"required,url"`
}

// JsonBatchRequestItem defines data in shortening response of batch of URLs.
type JsonBatchResponseItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id"`
	// Result is an alias value.
	Result string `json:"short_url"`
}

// JsonGetUserURLsResponseItem defines data in response to 'GetUserURLs' request.
type JsonGetUserURLsResponseItem struct {
	// ShortURL is an alias value.
	ShortURL string `json:"short_url"`
	// OriginalURL is an original URL value.
	OriginalURL string `json:"original_url"`
}
