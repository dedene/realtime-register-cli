package cmd

import (
	"testing"

	"github.com/dedene/realtime-register-cli/internal/api"
)

func TestFormatPeriods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		months []int
		want   string
	}{
		{nil, "-"},
		{[]int{12}, "1 year"},
		{[]int{24}, "2 years"},
		{[]int{12, 24}, "1, 2 years"},
		{[]int{12, 24, 36, 48, 60, 72, 84, 96, 108, 120}, "1-10 years"},
		{[]int{12, 60, 120}, "1, 5, 10 years"},
		{[]int{3, 12}, "3, 12 months"},
	}
	for _, tt := range tests {
		if got := formatPeriods(tt.months); got != tt.want {
			t.Errorf("formatPeriods(%v) = %q, want %q", tt.months, got, tt.want)
		}
	}
}

func TestFilterTLDs(t *testing.T) {
	t.Parallel()

	tlds := []api.TLDEntry{
		{TLD: "be", Provider: "Dnsbe"},
		{TLD: "brussels", Provider: "DnsbeBRUSSELS"},
		{TLD: "dev", Provider: "Google"},
		{TLD: "app", Provider: "Google"},
	}

	page, total := filterTLDs(tlds, ".DEV", 50, 0)
	if total != 1 || len(page) != 1 || page[0].TLD != "dev" {
		t.Errorf("search .DEV = %v (total %d), want [dev]", page, total)
	}

	page, total = filterTLDs(tlds, "google", 50, 0)
	if total != 2 || len(page) != 2 {
		t.Errorf("search google matched %d (total %d), want provider match on 2", len(page), total)
	}

	page, total = filterTLDs(tlds, "", 2, 1)
	if total != 4 || len(page) != 2 || page[0].TLD != "brussels" {
		t.Errorf("limit 2 offset 1 = %v (total %d), want [brussels dev]", page, total)
	}

	page, total = filterTLDs(tlds, "", 50, 10)
	if total != 4 || len(page) != 0 {
		t.Errorf("offset past end = %v (total %d), want empty", page, total)
	}
}
