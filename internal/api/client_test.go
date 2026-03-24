package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestMockServer_GetZone(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Close()

	zone := Zone{
		ID:   123,
		Name: "example.com",
		Records: []DNSRecord{
			{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
		},
	}

	mock.OnJSON("GET", "/dns/zones/123", 200, zone)

	client := mock.Client()
	got, err := client.GetZone(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}

	if got.ID != zone.ID {
		t.Errorf("GetZone().ID = %d, want %d", got.ID, zone.ID)
	}
	if got.Name != zone.Name {
		t.Errorf("GetZone().Name = %q, want %q", got.Name, zone.Name)
	}
	if len(got.Records) != 1 {
		t.Errorf("GetZone().Records = %d, want 1", len(got.Records))
	}
}

func TestMockServer_ListDomains(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Close()

	resp := ListResponse[Domain]{
		Entities: []Domain{
			{DomainName: "example.com", Status: []string{"ok"}},
			{DomainName: "example.org", Status: []string{"ok"}},
		},
		Pagination: Pagination{Limit: 50, Total: 2},
	}

	mock.OnJSON("GET", "/domains", 200, resp)

	client := mock.Client()
	got, err := client.ListDomains(context.Background(), DomainListOptions{})
	if err != nil {
		t.Fatalf("ListDomains() error = %v", err)
	}

	if len(got.Entities) != 2 {
		t.Errorf("ListDomains() = %d domains, want 2", len(got.Entities))
	}
}

func TestMockServer_NotFoundError(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Close()

	mock.OnJSON("GET", "/domains/notfound.com", 404, map[string]string{
		"message": "Domain not found",
	})

	client := mock.Client()
	_, err := client.GetDomain(context.Background(), "notfound.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected *NotFoundError, got %T", err)
	}
	if notFoundErr.StatusCode != 404 {
		t.Errorf("NotFoundError.StatusCode = %d, want 404", notFoundErr.StatusCode)
	}
}

func TestMockServer_APIError(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Close()

	mock.OnJSON("GET", "/domains/bad.com", 400, map[string]string{
		"message": "Bad request",
	})

	client := mock.Client()
	_, err := client.GetDomain(context.Background(), "bad.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("APIError.StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestUpdateZone_CorrelatesDnsRecordErrorsToRequestRecords(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.OnJSON(http.MethodPost, "/dns/zones/123/update", http.StatusBadRequest, map[string]any{
		"type":    "DnsConfigurationException",
		"message": "Errors have been found in the DNS records",
		"errors": map[string]any{
			"1": map[string]string{
				"content": "Must be a valid hostname",
			},
		},
		"conflicts": map[string]any{
			"1": []int{0},
		},
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{
		Records: []DNSRecord{
			{Type: "TXT", Name: "example.com", Content: "v=spf1 ~all", TTL: 3600},
			{Type: "MX", Name: "example.com", Content: "blackhole.tem.scaleway.com.", TTL: 3600, Prio: 10},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	wantParts := []string{
		`record 1.content: Must be a valid hostname [1: MX example.com -> blackhole.tem.scaleway.com. (prio 10)]`,
		`record 1 conflicts with records 0 [1: MX example.com -> blackhole.tem.scaleway.com. (prio 10); 0: TXT example.com -> v=spf1 ~all]`,
	}
	for _, want := range wantParts {
		if !strings.Contains(apiErr.Details, want) {
			t.Fatalf("Details = %q, want substring %q", apiErr.Details, want)
		}
	}
}

func TestUpdateZone_LeavesUnknownRecordIndexesUntouched(t *testing.T) {
	t.Parallel()

	mock := NewMockServer(t)
	defer mock.Close()

	mock.OnJSON(http.MethodPost, "/dns/zones/123/update", http.StatusBadRequest, map[string]any{
		"type":    "DnsConfigurationException",
		"message": "Errors have been found in the DNS records",
		"errors": map[string]any{
			"9": map[string]string{
				"content": "Must be a valid hostname",
			},
		},
	})

	client := mock.Client()
	err := client.UpdateZone(context.Background(), 123, &ZoneRequest{
		Records: []DNSRecord{
			{Type: "MX", Name: "example.com", Content: "mail.example.com", TTL: 3600, Prio: 10},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if got, want := apiErr.Details, "record 9.content: Must be a valid hostname"; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
}
