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

func TestUpdateZone_SendsPriorityForMXAndSRVRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record DNSRecord
	}{
		{
			name: "mx with zero priority",
			record: DNSRecord{
				Name:    "example.com",
				Type:    "MX",
				Content: "mail.example.com",
				TTL:     3600,
				Prio:    0,
			},
		},
		{
			name: "srv with zero priority",
			record: DNSRecord{
				Name:    "_sip._tcp.example.com",
				Type:    "SRV",
				Content: "sip.example.com",
				TTL:     3600,
				Prio:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := NewMockServer(t)
			defer mock.Close()

			mock.On(http.MethodPost, "/dns/zones/123/update", func(w http.ResponseWriter, r *http.Request) {
				defer func() { _ = r.Body.Close() }()

				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("Decode() error = %v", err)
				}

				records, ok := payload["records"].([]any)
				if !ok || len(records) != 1 {
					t.Fatalf("records = %#v, want single record", payload["records"])
				}

				record, ok := records[0].(map[string]any)
				if !ok {
					t.Fatalf("record = %#v, want object", records[0])
				}

				prio, ok := record["prio"]
				if !ok {
					t.Fatalf("record prio missing from payload: %#v", record)
				}
				if prio != float64(0) {
					t.Fatalf("record prio = %#v, want 0", prio)
				}

				w.WriteHeader(http.StatusOK)
			})

			client := mock.Client()
			err := client.UpdateZone(context.Background(), 123, &ZoneRequest{
				Records: []DNSRecord{tt.record},
			})
			if err != nil {
				t.Fatalf("UpdateZone() error = %v", err)
			}
		})
	}
}

func TestUpdateZone_OmitsPriorityForNonMXAndNonSRVRecords(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.On(http.MethodPost, "/dns/zones/123/update", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		records, ok := payload["records"].([]any)
		if !ok || len(records) != 1 {
			t.Fatalf("records = %#v, want single record", payload["records"])
		}

		record, ok := records[0].(map[string]any)
		if !ok {
			t.Fatalf("record = %#v, want object", records[0])
		}

		if _, ok := record["prio"]; ok {
			t.Fatalf("record prio unexpectedly present in payload: %#v", record)
		}

		w.WriteHeader(http.StatusOK)
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{
		Records: []DNSRecord{
			{
				Name:    "example.com",
				Type:    "TXT",
				Content: "v=spf1 include:_spf.example.com ~all",
				TTL:     3600,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}
}

func TestUpdateZone_UsesTTLFieldNameForZoneUpdates(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.On(http.MethodPost, "/dns/zones/123/update", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if got := payload["ttl"]; got != float64(1800) {
			t.Fatalf("ttl = %#v, want 1800", got)
		}
		if _, ok := payload["defaultTtl"]; ok {
			t.Fatalf("defaultTtl unexpectedly present in payload: %#v", payload)
		}
		if _, ok := payload["records"]; ok {
			t.Fatalf("records unexpectedly present in payload: %#v", payload)
		}

		w.WriteHeader(http.StatusOK)
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{TTL: 1800})
	if err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}
}

func TestUpdateZone_PreservesExplicitEmptyRecordSet(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.On(http.MethodPost, "/dns/zones/123/update", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		records, ok := payload["records"]
		if !ok {
			t.Fatalf("records missing from payload: %#v", payload)
		}

		items, ok := records.([]any)
		if !ok {
			t.Fatalf("records = %#v, want empty list", records)
		}
		if len(items) != 0 {
			t.Fatalf("len(records) = %d, want 0", len(items))
		}

		w.WriteHeader(http.StatusOK)
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{
		Records: []DNSRecord{},
	})
	if err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}
}

func TestUpdateZone_OmitsRecordsWhenNotProvided(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.On(http.MethodPost, "/dns/zones/123/update", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if _, ok := payload["records"]; ok {
			t.Fatalf("records unexpectedly present in payload: %#v", payload)
		}

		w.WriteHeader(http.StatusOK)
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{})
	if err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
