package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDomainListOptionsQueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		opts   DomainListOptions
		want   map[string]string
		absent []string
	}{
		{
			name:   "search maps to q, never search",
			opts:   DomainListOptions{ListOptions: ListOptions{Search: "acme", Limit: 10}},
			want:   map[string]string{"q": "acme", "limit": "10"},
			absent: []string{"search", "expiringWithin"},
		},
		{
			name:   "status and order pass through",
			opts:   DomainListOptions{ListOptions: ListOptions{Limit: 5}, Status: "ACTIVE", Order: "expiryDate"},
			want:   map[string]string{"limit": "5", "status": "ACTIVE", "order": "expiryDate"},
			absent: []string{"search", "expiringWithin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			values, err := url.ParseQuery(strings.TrimPrefix(tt.opts.QueryParams(), "?"))
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			for k, want := range tt.want {
				if got := values.Get(k); got != want {
					t.Errorf("query %q = %q, want %q", k, got, want)
				}
			}
			for _, k := range tt.absent {
				if values.Has(k) {
					t.Errorf("query unexpectedly contains %q (full: %q)", k, tt.opts.QueryParams())
				}
			}
		})
	}
}

func TestListExpiringDomains(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	// Ascending by expiryDate: one already expired, two within 30 days, one beyond.
	all := []Domain{
		{DomainName: "expired.com", ExpiryDate: now.AddDate(0, 0, -5)},
		{DomainName: "soon1.com", ExpiryDate: now.AddDate(0, 0, 3)},
		{DomainName: "soon2.com", ExpiryDate: now.AddDate(0, 0, 20)},
		{DomainName: "later.com", ExpiryDate: now.AddDate(0, 0, 90)},
	}

	mock := NewMockServer(t)
	mock.On(http.MethodGet, "/domains", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("order"); got != "expiryDate" {
			t.Fatalf("order = %q, want expiryDate", got)
		}
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		end := min(offset+limit, len(all))
		var page []Domain
		if offset < len(all) {
			page = all[offset:end]
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListResponse[Domain]{
			Entities:   page,
			Pagination: Pagination{Total: len(all), Offset: offset, Limit: limit},
		})
	})

	got, capped, err := mock.Client().ListExpiringDomains(context.Background(), 30, now)
	if err != nil {
		t.Fatalf("ListExpiringDomains() error = %v", err)
	}
	if capped {
		t.Fatalf("ListExpiringDomains() capped = true, want false")
	}
	if len(got) != 2 {
		t.Fatalf("ListExpiringDomains() returned %d domains, want 2: %+v", len(got), got)
	}
	if got[0].DomainName != "soon1.com" || got[1].DomainName != "soon2.com" {
		t.Fatalf("ListExpiringDomains() = %v, want [soon1.com soon2.com]", []string{got[0].DomainName, got[1].DomainName})
	}
}
