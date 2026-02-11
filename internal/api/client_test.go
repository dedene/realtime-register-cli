package api

import (
	"context"
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

	notFoundErr, ok := err.(*NotFoundError)
	if !ok {
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

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("APIError.StatusCode = %d, want 400", apiErr.StatusCode)
	}
}
