package api

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
)

// TLD API endpoints:
// GET /v2/providers        → ListProviders (the API has no plain TLD list; providers carry their TLDs)
// GET /v2/tlds/{tld}/info  → GetTLDInfo

// providerPageSize is the maximum page size /providers accepts.
const providerPageSize = 250

// Provider is a registry (or reseller) and the TLDs it serves.
type Provider struct {
	Name         string `json:"name"`
	ProviderType string `json:"providerType"`
	TLDs         []struct {
		Name string `json:"name"`
	} `json:"tlds"`
}

// TLDEntry is one TLD with the provider that serves it.
type TLDEntry struct {
	TLD      string `json:"tld"`
	Provider string `json:"provider"`
}

// TLDInfo is the /tlds/{tld}/info payload. Periods are in months.
type TLDInfo struct {
	Provider      string      `json:"provider"`
	ApplicableFor []string    `json:"applicableFor"`
	Metadata      TLDMetadata `json:"metadata"`

	raw json.RawMessage
}

// TLDMetadata holds the subset of TLD metadata the CLI displays. Periods are in months
// and grace periods in days.
type TLDMetadata struct {
	CreateDomainPeriods    []int    `json:"createDomainPeriods"`
	RenewDomainPeriods     []int    `json:"renewDomainPeriods"`
	TransferDomainPeriods  []int    `json:"transferDomainPeriods"`
	AutoRenewDomainPeriods []int    `json:"autoRenewDomainPeriods"`
	AddGracePeriod         int      `json:"addGracePeriod"`
	RenewGracePeriod       int      `json:"renewGracePeriod"`
	AutoRenewGracePeriod   int      `json:"autoRenewGracePeriod"`
	TransferGracePeriod    int      `json:"transferGracePeriod"`
	RedemptionPeriod       int      `json:"redemptionPeriod"`
	PendingDeletePeriod    int      `json:"pendingDeletePeriod"`
	FeaturesAvailable      []string `json:"featuresAvailable"`
	PremiumSupport         string   `json:"premiumSupport"`
	Jurisdiction           string   `json:"jurisdiction"`
	DomainSyntax           struct {
		MinLength  int  `json:"minLength"`
		MaxLength  int  `json:"maxLength"`
		IDNSupport bool `json:"idnSupport"`
	} `json:"domainSyntax"`
}

// UnmarshalJSON keeps the raw payload so --json output shows every field the API returns.
func (t *TLDInfo) UnmarshalJSON(data []byte) error {
	type plain TLDInfo
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*t = TLDInfo(p)
	t.raw = append(json.RawMessage(nil), data...)
	return nil
}

// MarshalJSON re-emits the raw API payload when available.
func (t *TLDInfo) MarshalJSON() ([]byte, error) {
	if len(t.raw) > 0 {
		return t.raw, nil
	}
	type plain TLDInfo
	return json.Marshal((*plain)(t))
}

// ListProviders returns one page of providers.
func (c *Client) ListProviders(ctx context.Context, limit, offset int) (*ListResponse[Provider], error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}
	var resp ListResponse[Provider]
	if err := c.Get(ctx, "/providers?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTLDs returns every supported TLD, sorted by name, by walking all provider pages.
func (c *Client) ListTLDs(ctx context.Context) ([]TLDEntry, error) {
	tlds := make([]TLDEntry, 0)
	for offset := 0; ; offset += providerPageSize {
		resp, err := c.ListProviders(ctx, providerPageSize, offset)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Entities {
			for _, t := range p.TLDs {
				tlds = append(tlds, TLDEntry{TLD: t.Name, Provider: p.Name})
			}
		}
		total := resp.Pagination.Total
		if len(resp.Entities) < providerPageSize || (total > 0 && offset+len(resp.Entities) >= total) {
			break
		}
	}
	sort.Slice(tlds, func(i, j int) bool { return tlds[i].TLD < tlds[j].TLD })
	return tlds, nil
}

// GetTLDInfo returns registry metadata for a single TLD.
func (c *Client) GetTLDInfo(ctx context.Context, tld string) (*TLDInfo, error) {
	var info TLDInfo
	if err := c.Get(ctx, "/tlds/"+url.PathEscape(tld)+"/info", &info); err != nil {
		return nil, err
	}
	return &info, nil
}
