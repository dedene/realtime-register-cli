package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestListTLDs_WalksAllProviderPagesAndSorts(t *testing.T) {
	t.Parallel()

	// Two full pages would need 500 providers; instead serve one full page then a short one.
	first := make([]map[string]any, providerPageSize)
	for i := range first {
		first[i] = map[string]any{"name": "P" + strconv.Itoa(i), "tlds": []map[string]string{{"name": "z" + strconv.Itoa(i)}}}
	}
	first[0]["tlds"] = []map[string]string{{"name": "dev"}, {"name": "app"}}
	second := []map[string]any{{"name": "Dnsbe", "tlds": []map[string]string{{"name": "be"}}}}

	var offsets []string
	mock := NewMockServer(t)
	defer mock.Close()
	mock.On(http.MethodGet, "/providers", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != strconv.Itoa(providerPageSize) {
			t.Errorf("limit = %q, want %d", q.Get("limit"), providerPageSize)
		}
		offsets = append(offsets, q.Get("offset"))
		page := first
		if q.Get("offset") == strconv.Itoa(providerPageSize) {
			page = second
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities":   page,
			"pagination": map[string]int{"total": providerPageSize + 1, "limit": providerPageSize},
		})
	})

	tlds, err := mock.Client().ListTLDs(context.Background())
	if err != nil {
		t.Fatalf("ListTLDs() error = %v", err)
	}
	if got := strings.Join(offsets, ","); got != ",250" {
		t.Errorf("requested offsets %q, want first page then 250", got)
	}
	if len(tlds) != providerPageSize+2 {
		t.Fatalf("len = %d, want %d", len(tlds), providerPageSize+2)
	}
	if tlds[0] != (TLDEntry{TLD: "app", Provider: "P0"}) || tlds[1] != (TLDEntry{TLD: "be", Provider: "Dnsbe"}) {
		t.Errorf("first entries = %+v, want app/P0 then be/Dnsbe (sorted)", tlds[:2])
	}
}

func TestGetTLDInfo_UsesInfoEndpointAndKeepsRawJSON(t *testing.T) {
	t.Parallel()

	body := `{"provider":"Google","applicableFor":["dev"],"metadata":{"createDomainPeriods":[12,24],` +
		`"redemptionPeriod":30,"jurisdiction":"US","domainSyntax":{"minLength":1,"maxLength":63,"idnSupport":true}}}`

	mock := NewMockServer(t)
	defer mock.Close()
	mock.On(http.MethodGet, "/tlds/dev/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	info, err := mock.Client().GetTLDInfo(context.Background(), "dev")
	if err != nil {
		t.Fatalf("GetTLDInfo() error = %v", err)
	}
	if info.Provider != "Google" || info.Metadata.RedemptionPeriod != 30 || !info.Metadata.DomainSyntax.IDNSupport {
		t.Errorf("decoded = %+v", *info)
	}
	if got := info.Metadata.CreateDomainPeriods; len(got) != 2 || got[1] != 24 {
		t.Errorf("CreateDomainPeriods = %v, want [12 24]", got)
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	// "jurisdiction" round-trips, but so would any field the CLI does not model.
	if string(out) != body {
		t.Errorf("Marshal() = %s, want raw payload %s", out, body)
	}
}
