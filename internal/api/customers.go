package api

import (
	"context"
	"net/url"
)

// Customer API endpoints:
// GET /v2/customers/{customer}/pricelist → GetPricelist

// GetPricelist returns the customer's pricelist.
func (c *Client) GetPricelist(ctx context.Context, customer string) (*Pricelist, error) {
	var resp Pricelist
	path := "/customers/" + url.PathEscape(customer) + "/pricelist"
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
