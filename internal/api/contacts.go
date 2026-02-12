package api

import (
	"context"
	"fmt"
	"net/url"
)

// Contact API endpoints per SPEC.md:
// GET    /v2/customers/{customer}/contacts           → ListContacts
// GET    /v2/customers/{customer}/contacts/{handle}  → GetContact
// POST   /v2/customers/{customer}/contacts/{handle}  → CreateContact
// POST   /v2/customers/{customer}/contacts/{handle}/update → UpdateContact
// DELETE /v2/customers/{customer}/contacts/{handle}  → DeleteContact

// ContactRequest for creating/updating contacts.
type ContactRequest struct {
	Name         string   `json:"name"`
	Organization string   `json:"organization,omitempty"`
	Email        string   `json:"email"`
	Phone        string   `json:"voice"`
	Fax          string   `json:"fax,omitempty"`
	Address      []string `json:"addressLine"`
	City         string   `json:"city"`
	State        string   `json:"state,omitempty"`
	PostalCode   string   `json:"postalCode"`
	Country      string   `json:"country"` // ISO 2-letter code
}

// ContactListOptions for filtering contacts.
type ContactListOptions struct {
	ListOptions
}

// QueryParams builds a URL query string from contact list options.
func (o ContactListOptions) QueryParams() string {
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
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ListContacts returns contacts for a customer.
func (c *Client) ListContacts(ctx context.Context, customer string, opts ContactListOptions) (*ListResponse[Contact], error) {
	var resp ListResponse[Contact]
	path := fmt.Sprintf("/customers/%s/contacts%s", url.PathEscape(customer), opts.QueryParams())
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetContact returns a single contact.
func (c *Client) GetContact(ctx context.Context, customer, handle string) (*Contact, error) {
	var contact Contact
	path := fmt.Sprintf("/customers/%s/contacts/%s", url.PathEscape(customer), url.PathEscape(handle))
	if err := c.Get(ctx, path, &contact); err != nil {
		return nil, err
	}
	return &contact, nil
}

// CreateContact creates a new contact.
func (c *Client) CreateContact(ctx context.Context, customer, handle string, req *ContactRequest) error {
	path := fmt.Sprintf("/customers/%s/contacts/%s", url.PathEscape(customer), url.PathEscape(handle))
	return c.Post(ctx, path, req, nil)
}

// UpdateContact updates an existing contact.
func (c *Client) UpdateContact(ctx context.Context, customer, handle string, req *ContactRequest) error {
	path := fmt.Sprintf("/customers/%s/contacts/%s/update", url.PathEscape(customer), url.PathEscape(handle))
	return c.Post(ctx, path, req, nil)
}

// DeleteContact deletes a contact.
func (c *Client) DeleteContact(ctx context.Context, customer, handle string) error {
	path := fmt.Sprintf("/customers/%s/contacts/%s", url.PathEscape(customer), url.PathEscape(handle))
	return c.Delete(ctx, path)
}
