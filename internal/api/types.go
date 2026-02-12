package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ListResponse is the paginated response wrapper.
type ListResponse[T any] struct {
	Entities   []T        `json:"entities"`
	Pagination Pagination `json:"pagination"`
}

// Pagination holds offset-based pagination metadata.
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// ListOptions for paginated requests.
type ListOptions struct {
	Limit  int
	Offset int
	Search string
}

// QueryParams builds a URL query string from list options.
func (o ListOptions) QueryParams() string {
	v := url.Values{}
	if o.Limit > 0 {
		v.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	if o.Search != "" {
		v.Set("q", o.Search)
	}
	if encoded := v.Encode(); encoded != "" {
		return "?" + encoded
	}
	return ""
}

// Domain represents a domain registration.
type Domain struct {
	DomainName      string    `json:"domainName"`
	Registry        string    `json:"registry"`
	Customer        string    `json:"customer"`
	Status          []string  `json:"status"`
	ExpiryDate      time.Time `json:"expiryDate"`
	AutoRenew       bool      `json:"autoRenew"`
	AutoRenewPeriod int       `json:"autoRenewPeriod"`
	Registrant      string    `json:"registrant"`
	NameServers     []string  `json:"ns"`
	AuthCode        string    `json:"authcode,omitempty"`
	CreatedDate     time.Time `json:"createdDate"`
	UpdatedDate     time.Time `json:"updatedDate,omitempty"`
	PrivacyProtect  bool      `json:"privacyProtect"`
	Premium         bool      `json:"premium"`
	BillingHandle   string    `json:"billingHandle,omitempty"`
	TechHandle      string    `json:"techHandle,omitempty"`
	AdminHandle     string    `json:"adminHandle,omitempty"`
}

func (d Domain) String() string {
	status := "unknown"
	if len(d.Status) > 0 {
		status = d.Status[0]
	}
	return fmt.Sprintf("%s (%s, expires %s)", d.DomainName, status, d.ExpiryDate.Format("2006-01-02"))
}

// Contact represents a contact handle.
type Contact struct {
	Handle       string    `json:"handle"`
	Customer     string    `json:"customer"`
	Brand        string    `json:"brand,omitempty"`
	Name         string    `json:"name"`
	Organization string    `json:"organization,omitempty"`
	Email        string    `json:"email"`
	Phone        string    `json:"voice"`
	Fax          string    `json:"fax,omitempty"`
	AddressLine  []string  `json:"addressLine"`
	PostalCode   string    `json:"postalCode"`
	City         string    `json:"city"`
	State        string    `json:"state,omitempty"`
	Country      string    `json:"country"`
	CreatedDate  time.Time `json:"createdDate"`
}

// Zone represents a DNS zone.
type Zone struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Customer    string      `json:"customer"`
	Service     string      `json:"service,omitempty"`
	TTL         int         `json:"ttl"`
	DNSSec      bool        `json:"dnssec"`
	Records     []DNSRecord `json:"records,omitempty"`
	CreatedDate time.Time   `json:"createdDate"`
	UpdatedDate time.Time   `json:"updatedDate,omitempty"`
}

// DNSRecord represents a DNS resource record.
type DNSRecord struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Prio    int    `json:"prio,omitempty"`
}

// Process represents an async operation.
type Process struct {
	ID           int       `json:"id"`
	User         string    `json:"user"`
	Customer     string    `json:"customer"`
	Status       string    `json:"status"`
	StatusDetail string    `json:"statusDetail,omitempty"`
	Action       string    `json:"action"`
	Type         string    `json:"type"`
	Entity       string    `json:"entity,omitempty"`
	Identifier   string    `json:"identifier,omitempty"`
	CreatedDate  time.Time `json:"createdDate"`
	UpdatedDate  time.Time `json:"updatedDate,omitempty"`
	StartedDate  time.Time `json:"startedDate,omitempty"`
	Message      string    `json:"message,omitempty"`
}

// TLDInfo represents top-level domain metadata.
type TLDInfo struct {
	TLD           string  `json:"tld"`
	PriceCreate   float64 `json:"priceCreate"`
	PriceRenew    float64 `json:"priceRenew"`
	PriceTransfer float64 `json:"priceTransfer"`
	MinPeriod     int     `json:"minPeriod"`
	MaxPeriod     int     `json:"maxPeriod"`
}

// PricelistEntry represents a price from the customer pricelist.
type PricelistEntry struct {
	Product  string `json:"product"`
	Action   string `json:"action"`
	Currency string `json:"currency"`
	Price    int    `json:"price"` // in cents
}

// Pricelist is the response from the pricelist endpoint.
type Pricelist struct {
	Prices []PricelistEntry `json:"prices"`
}

// GetTLDPrice finds the CREATE price for a TLD in cents, returns price and currency.
func (p *Pricelist) GetTLDPrice(tld string) (price int, currency string, found bool) {
	product := "domain_" + tld
	for _, entry := range p.Prices {
		if entry.Product == product && entry.Action == "CREATE" {
			return entry.Price, entry.Currency, true
		}
	}
	return 0, "", false
}
