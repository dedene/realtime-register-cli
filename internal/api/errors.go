package api

import (
	"encoding/json"
	"fmt"
	"time"
)

// APIError represents a generic API error response.
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}

// AuthError represents a 401/403 authentication error.
type AuthError struct {
	APIError
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

// RateLimitError represents a 429 rate limit error.
type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: retry after %ds", int(e.RetryAfter.Seconds()))
}

// NotFoundError represents a 404 resource not found error.
type NotFoundError struct {
	APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Message)
}

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s %s", e.Field, e.Message)
}

// apiErrorResponse mirrors the RealtimeRegister JSON error format.
// {
//
//	"error": {
//	    "code": 400,
//	    "message": "Domain not found"
//	}
//
// }
type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// NewAPIError parses a JSON error response body and returns the appropriate typed error.
func NewAPIError(statusCode int, body []byte) error {
	var resp apiErrorResponse
	msg := "unknown error"
	if err := json.Unmarshal(body, &resp); err == nil && resp.Error.Message != "" {
		msg = resp.Error.Message
	}

	base := APIError{
		StatusCode: statusCode,
		Message:    msg,
	}

	switch statusCode {
	case 401, 403:
		return &AuthError{APIError: base}
	case 404:
		return &NotFoundError{APIError: base}
	case 429:
		return &RateLimitError{APIError: base, RetryAfter: parseRetryAfter(body)}
	default:
		return &base
	}
}

// parseRetryAfter attempts to extract retry-after seconds from the response body.
func parseRetryAfter(body []byte) time.Duration {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0
	}

	if v, ok := raw["retry_after"]; ok {
		if val, ok := v.(float64); ok {
			return time.Duration(val) * time.Second
		}
	}
	return 0
}
