// Package handlers contains definitions that are used in handlers of both
// http and gRPC types.
package handlers

// JSONShortenRequest defines data in shortening request (JSON format).
type JSONShortenRequest struct {
	// URL is a URL for shortening.
	URL string `json:"url" validate:"required,url"`
}

// JSONShortenResponse defines data in shortening response (JSON format).
type JSONShortenResponse struct {
	// Result is an alias value.
	Result string `json:"result"`
}

// JSONBatchRequestItem defines data in shortening request of batch of URLs.
type JSONBatchRequestItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id" validate:"required"`
	// Original is a URL for shortening.
	Original string `json:"original_url" validate:"required,url"`
}

// JSONBatchRequestItem defines data in shortening response of batch of URLs.
type JSONBatchResponseItem struct {
	// ID is an identidier of URL for shortening.
	ID string `json:"correlation_id"`
	// Result is an alias value.
	Result string `json:"short_url"`
}

// JSONGetUserURLsResponseItem defines data in response to 'GetUserURLs' request.
type JSONGetUserURLsResponseItem struct {
	// ShortURL is an alias value.
	ShortURL string `json:"short_url"`
	// OriginalURL is an original URL value.
	OriginalURL string `json:"original_url"`
}
