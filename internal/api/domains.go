package api

import (
	"context"
	"fmt"
	"net/url"
)

// Domain API endpoints per SPEC.md Appendix A:
// GET    /v2/domains              → ListDomains
// GET    /v2/domains/{name}       → GetDomain
// GET    /v2/domains/{name}/check → CheckDomain
// POST   /v2/domains/{name}       → RegisterDomain
// POST   /v2/domains/{name}/update → UpdateDomain
// DELETE /v2/domains/{name}       → DeleteDomain
// POST   /v2/domains/{name}/renew → RenewDomain
// POST   /v2/domains/{name}/transfer → TransferDomain

// DomainAvailability is the response from domain check.
type DomainAvailability struct {
	Available bool    `json:"available"`
	Domain    string  `json:"domain"`
	Premium   bool    `json:"premium,omitempty"`
	Price     float64 `json:"price,omitempty"`
}

// RegisterRequest for domain registration.
type RegisterRequest struct {
	Period       int      `json:"period,omitempty"`
	Registrant   string   `json:"registrant"`
	Admin        string   `json:"admin,omitempty"`
	Tech         string   `json:"tech,omitempty"`
	Billing      string   `json:"billing,omitempty"`
	Nameservers  []string `json:"ns,omitempty"`
	AutoRenew    *bool    `json:"autoRenew,omitempty"`
	PrivacyProxy *bool    `json:"privacyProxy,omitempty"`
}

// UpdateRequest for domain updates.
type UpdateRequest struct {
	Registrant  string   `json:"registrant,omitempty"`
	Admin       string   `json:"admin,omitempty"`
	Tech        string   `json:"tech,omitempty"`
	Billing     string   `json:"billing,omitempty"`
	Nameservers []string `json:"ns,omitempty"`
	AutoRenew   *bool    `json:"autoRenew,omitempty"`
}

// RenewRequest for domain renewal.
type RenewRequest struct {
	Period int `json:"period"`
}

// TransferRequest for domain transfer.
type TransferRequest struct {
	AuthCode   string `json:"authCode"`
	Registrant string `json:"registrant,omitempty"`
	AutoRenew  *bool  `json:"autoRenew,omitempty"`
}

// DomainListOptions extends ListOptions for domain-specific filters.
type DomainListOptions struct {
	ListOptions
	Status         string
	ExpiringWithin int
	Order          string
}

// QueryParams builds a URL query string from domain list options.
func (o DomainListOptions) QueryParams() string {
	v := url.Values{}
	if o.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	if o.Search != "" {
		v.Set("search", o.Search)
	}
	if o.Status != "" {
		v.Set("status", o.Status)
	}
	if o.ExpiringWithin > 0 {
		v.Set("expiringWithin", fmt.Sprintf("%d", o.ExpiringWithin))
	}
	if o.Order != "" {
		v.Set("order", o.Order)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ListDomains returns paginated domain list.
func (c *Client) ListDomains(ctx context.Context, opts DomainListOptions) (*ListResponse[Domain], error) {
	var resp ListResponse[Domain]
	if err := c.Get(ctx, "/domains"+opts.QueryParams(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDomain returns a single domain.
func (c *Client) GetDomain(ctx context.Context, name string) (*Domain, error) {
	var domain Domain
	if err := c.Get(ctx, "/domains/"+url.PathEscape(name), &domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

// CheckDomain checks availability.
func (c *Client) CheckDomain(ctx context.Context, name string) (*DomainAvailability, error) {
	var result DomainAvailability
	if err := c.Get(ctx, "/domains/"+url.PathEscape(name)+"/check", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RegisterDomain registers a new domain.
func (c *Client) RegisterDomain(ctx context.Context, name string, req RegisterRequest) (*Process, error) {
	var process Process
	if err := c.Post(ctx, "/domains/"+url.PathEscape(name), req, &process); err != nil {
		return nil, err
	}
	return &process, nil
}

// UpdateDomain updates domain settings.
func (c *Client) UpdateDomain(ctx context.Context, name string, req UpdateRequest) error {
	return c.Post(ctx, "/domains/"+url.PathEscape(name)+"/update", req, nil)
}

// DeleteDomain deletes a domain.
func (c *Client) DeleteDomain(ctx context.Context, name string) error {
	return c.Delete(ctx, "/domains/"+url.PathEscape(name))
}

// RenewDomain renews a domain.
func (c *Client) RenewDomain(ctx context.Context, name string, period int) (*Process, error) {
	var process Process
	req := RenewRequest{Period: period}
	if err := c.Post(ctx, "/domains/"+url.PathEscape(name)+"/renew", req, &process); err != nil {
		return nil, err
	}
	return &process, nil
}

// TransferDomain initiates a domain transfer.
func (c *Client) TransferDomain(ctx context.Context, name string, req TransferRequest) (*Process, error) {
	var process Process
	if err := c.Post(ctx, "/domains/"+url.PathEscape(name)+"/transfer", req, &process); err != nil {
		return nil, err
	}
	return &process, nil
}
