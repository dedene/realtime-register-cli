package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestZoneListOptionsQueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts ZoneListOptions
		want url.Values
	}{
		{
			name: "search and pagination",
			opts: ZoneListOptions{
				ListOptions: ListOptions{
					Limit:  50,
					Offset: 10,
					Search: "example",
				},
			},
			want: url.Values{
				"limit":  []string{"50"},
				"offset": []string{"10"},
				"q":      []string{"example"},
			},
		},
		{
			name: "common filters",
			opts: ZoneListOptions{
				ListOptions: ListOptions{
					Limit: 2,
				},
				Name:    "example.com",
				Managed: boolPtr(true),
				Service: "PREMIUM",
			},
			want: url.Values{
				"limit":   []string{"2"},
				"name":    []string{"example.com"},
				"managed": []string{"true"},
				"service": []string{"PREMIUM"},
			},
		},
		{
			name: "unmanaged filter",
			opts: ZoneListOptions{
				Managed: boolPtr(false),
			},
			want: url.Values{
				"managed": []string{"false"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := tt.opts.QueryParams()
			got, err := url.ParseQuery(raw[1:])
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			if got.Encode() != tt.want.Encode() {
				t.Fatalf("QueryParams() = %q, want %q", got.Encode(), tt.want.Encode())
			}
		})
	}
}

func TestMockServer_GetZoneByDomain(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	zone := Zone{ID: 123, Name: "example.com"}
	mock.OnJSON(http.MethodGet, "/domains/example.com/zone", http.StatusOK, zone)

	client := mock.Client()
	got, err := client.GetZoneByDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("GetZoneByDomain() error = %v", err)
	}
	if got.ID != zone.ID {
		t.Fatalf("GetZoneByDomain().ID = %d, want %d", got.ID, zone.ID)
	}
}

func TestMockServer_ListZonesIncludesFilters(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.On(http.MethodGet, "/dns/zones", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("name"); got != "example.com" {
			t.Fatalf("name filter = %q, want %q", got, "example.com")
		}
		if got := q.Get("managed"); got != "false" {
			t.Fatalf("managed filter = %q, want %q", got, "false")
		}
		if got := q.Get("service"); got != "BASIC" {
			t.Fatalf("service filter = %q, want %q", got, "BASIC")
		}
		if got := q.Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want %q", got, "2")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListResponse[Zone]{Entities: []Zone{{ID: 1, Name: "example.com"}}})
	})

	client := mock.Client()
	_, err := client.ListZones(context.Background(), ZoneListOptions{
		ListOptions: ListOptions{Limit: 2},
		Name:        "example.com",
		Managed:     boolPtr(false),
		Service:     "BASIC",
	})
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
