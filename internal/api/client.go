package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
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
		err := NewAPIError(resp.StatusCode, respBody)
		return correlateAPIError(err, body)
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

var (
	recordIndexPattern  = regexp.MustCompile(`\brecord (\d+)`)
	recordsIndexPattern = regexp.MustCompile(`\brecords ([0-9,\s]+)`)
)

func correlateAPIError(err error, requestBody any) error {
	if requestBody == nil {
		return err
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Details == "" {
		return err
	}

	summaries := extractRecordSummaries(requestBody)
	if len(summaries) == 0 {
		return err
	}

	apiErr.Details = correlateRecordDetails(apiErr.Details, summaries)
	return err
}

func extractRecordSummaries(requestBody any) map[string]string {
	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	items, ok := raw["records"].([]any)
	if !ok {
		return nil
	}

	summaries := make(map[string]string, len(items))
	for i, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		summaries[strconv.Itoa(i)] = summarizeRecord(record)
	}
	return summaries
}

func summarizeRecord(record map[string]any) string {
	typ := stringValue(record["type"])
	name := stringValue(record["name"])
	content := stringValue(record["content"])
	summary := strings.TrimSpace(typ + " " + name)
	if content != "" {
		summary = strings.TrimSpace(summary + " -> " + content)
	}
	if prio := stringValue(record["prio"]); prio != "" {
		summary = strings.TrimSpace(summary + " (prio " + prio + ")")
	}
	return summary
}

func correlateRecordDetails(details string, summaries map[string]string) string {
	lines := strings.Split(details, "\n")
	for i, line := range lines {
		references := referencedRecordIndexes(line)
		if len(references) == 0 {
			continue
		}

		parts := make([]string, 0, len(references))
		for _, index := range references {
			summary, ok := summaries[index]
			if !ok || summary == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s: %s", index, summary))
		}
		if len(parts) == 0 {
			continue
		}

		lines[i] = line + " [" + strings.Join(parts, "; ") + "]"
	}

	return strings.Join(lines, "\n")
}

func referencedRecordIndexes(line string) []string {
	var indexes []string

	for _, match := range recordIndexPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 2 {
			continue
		}
		indexes = append(indexes, match[1])
	}

	for _, match := range recordsIndexPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 2 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				indexes = append(indexes, part)
			}
		}
	}

	return uniqueNonEmpty(indexes)
}
