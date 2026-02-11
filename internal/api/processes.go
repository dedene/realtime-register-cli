package api

import (
	"context"
	"fmt"
	"net/url"
)

// Process API endpoints per SPEC.md:
// GET    /v2/processes       → ListProcesses
// GET    /v2/processes/{id}  → GetProcess
// GET    /v2/processes/{id}/info → GetProcessInfo
// DELETE /v2/processes/{id}  → CancelProcess
// POST   /v2/processes/{id}/resend → ResendProcess

// ProcessListOptions for filtering processes.
type ProcessListOptions struct {
	ListOptions
	Status string
}

// QueryParams builds a URL query string from process list options.
func (o ProcessListOptions) QueryParams() string {
	v := url.Values{}
	if o.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ProcessInfo contains extended process information.
type ProcessInfo struct {
	Process
	Details map[string]any `json:"details,omitempty"`
}

// ListProcesses returns paginated process list.
func (c *Client) ListProcesses(ctx context.Context, opts ProcessListOptions) (*ListResponse[Process], error) {
	var resp ListResponse[Process]
	if err := c.Get(ctx, "/processes"+opts.QueryParams(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProcess returns a single process.
func (c *Client) GetProcess(ctx context.Context, id int) (*Process, error) {
	var process Process
	if err := c.Get(ctx, fmt.Sprintf("/processes/%d", id), &process); err != nil {
		return nil, err
	}
	return &process, nil
}

// GetProcessInfo returns extended process info.
func (c *Client) GetProcessInfo(ctx context.Context, id int) (*ProcessInfo, error) {
	var info ProcessInfo
	if err := c.Get(ctx, fmt.Sprintf("/processes/%d/info", id), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// CancelProcess cancels a pending process.
func (c *Client) CancelProcess(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/processes/%d", id))
}

// ResendProcess resends notifications for a process.
func (c *Client) ResendProcess(ctx context.Context, id int) error {
	return c.Post(ctx, fmt.Sprintf("/processes/%d/resend", id), nil, nil)
}
