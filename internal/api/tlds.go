package api

import (
	"context"
	"net/url"
)

// TLD API endpoints:
// GET /v2/tlds       → ListTLDs
// GET /v2/tlds/{tld} → GetTLD

// TLDListOptions for filtering TLDs.
type TLDListOptions struct {
	ListOptions
}

// ListTLDs returns available TLDs.
func (c *Client) ListTLDs(ctx context.Context, opts TLDListOptions) (*ListResponse[TLDInfo], error) {
	var resp ListResponse[TLDInfo]
	if err := c.Get(ctx, "/tlds"+opts.QueryParams(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTLD returns info for a single TLD.
func (c *Client) GetTLD(ctx context.Context, tld string) (*TLDInfo, error) {
	var info TLDInfo
	if err := c.Get(ctx, "/tlds/"+url.PathEscape(tld), &info); err != nil {
		return nil, err
	}
	return &info, nil
}
