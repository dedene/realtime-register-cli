package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ProductionURL  = "https://api.yoursrs.com/v2"
	SandboxURL     = "https://api.yoursrs-ote.com/v2"
	defaultTimeout = 30 * time.Second
)

// Client is the Realtime Register API client.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	userAgent  string
}

// NewClient creates a Client with retry transport and auth.
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: NewRetryTransport(http.DefaultTransport),
			Timeout:   defaultTimeout,
		},
		apiKey:    apiKey,
		baseURL:   ProductionURL,
		userAgent: "rr/dev",
	}
}

// SetBaseURL overrides the API base URL.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// SetUserAgent overrides the User-Agent header.
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}

// do executes an API request with auth and error handling.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NewAPIError(resp.StatusCode, respBody)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}
