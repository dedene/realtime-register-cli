package api

import (
	"context"
	"fmt"
	"net/url"
)

// Zone API endpoints per SPEC.md:
// GET    /v2/dns/zones      → ListZones
// GET    /v2/dns/zones/{id} → GetZone
// POST   /v2/dns/zones      → CreateZone
// POST   /v2/dns/zones/{id}/update → UpdateZone
// DELETE /v2/dns/zones/{id} → DeleteZone

// ZoneRequest for creating/updating zones.
type ZoneRequest struct {
	Name       string      `json:"name,omitempty"`
	TTL        int         `json:"defaultTtl,omitempty"`
	Refresh    int         `json:"refresh,omitempty"`
	Retry      int         `json:"retry,omitempty"`
	Expire     int         `json:"expire,omitempty"`
	DNSSecMode string      `json:"dnssecMode,omitempty"`
	Records    []DNSRecord `json:"records,omitempty"`
}

// CreateZoneResponse is returned when creating a zone.
type CreateZoneResponse struct {
	ID int `json:"id"`
}

// ZoneListOptions for filtering zones.
type ZoneListOptions struct {
	ListOptions
}

// QueryParams builds a URL query string from zone list options.
func (o ZoneListOptions) QueryParams() string {
	v := url.Values{}
	if o.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	if o.Search != "" {
		v.Set("q", o.Search)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ListZones returns paginated zone list.
func (c *Client) ListZones(ctx context.Context, opts ZoneListOptions) (*ListResponse[Zone], error) {
	var resp ListResponse[Zone]
	if err := c.Get(ctx, "/dns/zones"+opts.QueryParams(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetZone returns a single zone by ID.
func (c *Client) GetZone(ctx context.Context, id int) (*Zone, error) {
	var zone Zone
	if err := c.Get(ctx, fmt.Sprintf("/dns/zones/%d", id), &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// CreateZone creates a new DNS zone.
func (c *Client) CreateZone(ctx context.Context, req *ZoneRequest) (int, error) {
	var resp CreateZoneResponse
	if err := c.Post(ctx, "/dns/zones", req, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// UpdateZone updates a DNS zone.
func (c *Client) UpdateZone(ctx context.Context, id int, req *ZoneRequest) error {
	return c.Post(ctx, fmt.Sprintf("/dns/zones/%d/update", id), req, nil)
}

// DeleteZone deletes a DNS zone.
func (c *Client) DeleteZone(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/dns/zones/%d", id))
}
