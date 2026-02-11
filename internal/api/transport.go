package api

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

// retryableStatusCodes are HTTP status codes that trigger a retry.
var retryableStatusCodes = map[int]bool{
	429: true, // Rate limited
	500: true,
	502: true,
	503: true,
	504: true,
}

// RetryTransport wraps an http.RoundTripper with retry logic and exponential backoff.
type RetryTransport struct {
	Base       http.RoundTripper
	MaxRetries int
}

// NewRetryTransport creates a RetryTransport with default 3 retries.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RetryTransport{
		Base:       base,
		MaxRetries: 3,
	}
}

// RoundTrip executes the request with retry logic.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if !retryableStatusCodes[resp.StatusCode] {
			return resp, nil
		}

		// Last attempt -- return as-is.
		if attempt == t.MaxRetries {
			return resp, nil
		}

		// Compute backoff: 1s, 2s, 4s ...
		backoff := time.Second * (1 << uint(attempt))

		// Respect Retry-After header if present and larger.
		if ra := parseRetryAfterHeader(resp.Header.Get("Retry-After")); ra > backoff {
			backoff = ra
		}

		// Drain and close body before retry.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		// Wait for backoff or context cancellation.
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		}
	}

	return resp, nil
}

// parseRetryAfterHeader parses a Retry-After header value as either
// integer seconds or HTTP-date format.
func parseRetryAfterHeader(header string) time.Duration {
	if header == "" {
		return 0
	}

	// Try integer seconds first.
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try HTTP-date format.
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}

	return 0
}
