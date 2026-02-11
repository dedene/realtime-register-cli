package cmd

import (
	"testing"

	"github.com/dedene/realtime-register-cli/internal/api"
)

func TestFindRecords(t *testing.T) {
	records := []api.DNSRecord{
		{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
		{Type: "A", Name: "@", Content: "5.6.7.8", TTL: 3600},
		{Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600, Prio: 10},
		{Type: "TXT", Name: "_dmarc", Content: "v=DMARC1", TTL: 3600},
		{Type: "CNAME", Name: "www", Content: "example.com", TTL: 3600},
	}

	tests := []struct {
		name    string
		typ     string
		recName string
		content string
		want    int
	}{
		{"single A by content", "A", "@", "1.2.3.4", 1},
		{"all A records", "A", "@", "", 2},
		{"MX record", "MX", "@", "", 1},
		{"TXT record", "TXT", "_dmarc", "", 1},
		{"CNAME record", "CNAME", "www", "", 1},
		{"no match type", "AAAA", "@", "", 0},
		{"no match name", "A", "nonexistent", "", 0},
		{"case insensitive type", "a", "@", "", 2},
		{"content mismatch", "A", "@", "9.9.9.9", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findRecords(records, tt.typ, tt.recName, tt.content)
			if len(got) != tt.want {
				t.Errorf("findRecords(%q, %q, %q) returned %d indices, want %d",
					tt.typ, tt.recName, tt.content, len(got), tt.want)
			}
		})
	}
}

func TestFindRecords_ReturnsCorrectIndices(t *testing.T) {
	records := []api.DNSRecord{
		{Type: "A", Name: "@", Content: "1.2.3.4"},
		{Type: "MX", Name: "@", Content: "mail.example.com"},
		{Type: "A", Name: "@", Content: "5.6.7.8"},
	}

	// Should return indices 0 and 2 (the A records)
	indices := findRecords(records, "A", "@", "")
	if len(indices) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 2 {
		t.Errorf("expected indices [0, 2], got %v", indices)
	}

	// With content filter, should return only index 2
	indices = findRecords(records, "A", "@", "5.6.7.8")
	if len(indices) != 1 {
		t.Fatalf("expected 1 index, got %d", len(indices))
	}
	if indices[0] != 2 {
		t.Errorf("expected index 2, got %d", indices[0])
	}
}
