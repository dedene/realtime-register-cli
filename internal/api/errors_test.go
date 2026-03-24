package api

import (
	"errors"
	"strings"
	"testing"
)

func TestNewAPIError_ParsesTopLevelMessage(t *testing.T) {
	t.Parallel()

	err := NewAPIError(400, []byte(`{"message":"Bad request"}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if got, want := apiErr.Message, "Bad request"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}
}

func TestNewAPIError_ParsesNestedErrorDetails(t *testing.T) {
	t.Parallel()

	err := NewAPIError(400, []byte(`{"error":{"code":400,"message":"Validation failed","details":"record content invalid"}}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if got, want := apiErr.Message, "Validation failed"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}
	if got, want := apiErr.Details, "record content invalid"; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
}

func TestNewAPIError_FallsBackToRawBodyForUnknownShape(t *testing.T) {
	t.Parallel()

	err := NewAPIError(400, []byte(`{"violations":[{"field":"records[0].content","message":"invalid MX target"}]}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if got, want := apiErr.Message, "unknown error"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}
	if got, want := apiErr.Details, `{"violations":[{"field":"records[0].content","message":"invalid MX target"}]}`; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
}

func TestNewAPIError_ParsesDnsConfigurationExceptionDetails(t *testing.T) {
	t.Parallel()

	body := `{
		"type":"DnsConfigurationException",
		"message":"The DNS records contain errors",
		"errors":{
			"14":{"content":"invalid MX target"},
			"2":{"type":"unsupported record type"}
		},
		"conflicts":{
			"14":[0,3]
		}
	}`

	err := NewAPIError(400, []byte(body))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if got, want := apiErr.Message, "The DNS records contain errors"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}

	wantParts := []string{
		"record 2.type: unsupported record type",
		"record 14.content: invalid MX target",
		"record 14 conflicts with records 0, 3",
	}
	for _, want := range wantParts {
		if !strings.Contains(apiErr.Details, want) {
			t.Fatalf("Details = %q, want substring %q", apiErr.Details, want)
		}
	}
}

func TestNewAPIError_FlattensStringListFieldErrors(t *testing.T) {
	t.Parallel()

	body := `{
		"message":"Errors have been found in the DNS records",
		"errors":{
			"14":{"content":["Must be a valid hostname"]}
		}
	}`

	err := NewAPIError(400, []byte(body))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if got, want := apiErr.Details, "record 14.content: Must be a valid hostname"; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
}

func TestNewAPIError_SortsRecordIndexesNumerically(t *testing.T) {
	t.Parallel()

	body := `{
		"message":"Errors have been found in the DNS records",
		"errors":{
			"10":{"content":"ten"},
			"2":{"content":"two"}
		}
	}`

	err := NewAPIError(400, []byte(body))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	first := strings.Index(apiErr.Details, "record 2.content: two")
	second := strings.Index(apiErr.Details, "record 10.content: ten")
	if first == -1 || second == -1 {
		t.Fatalf("Details missing expected lines: %q", apiErr.Details)
	}
	if first > second {
		t.Fatalf("Details not numerically sorted: %q", apiErr.Details)
	}
}
