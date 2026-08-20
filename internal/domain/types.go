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
