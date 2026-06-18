package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestProcessListOptionsQueryParams(t *testing.T) {
	t.Parallel()

	t.Run("repeated status for multiple statuses", func(t *testing.T) {
		t.Parallel()

		opts := ProcessListOptions{
			ListOptions: ListOptions{Limit: 1},
			Statuses:    []string{"NEW", "RUNNING"},
		}
		values, err := url.ParseQuery(strings.TrimPrefix(opts.QueryParams(), "?"))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		got := values["status"]
		if len(got) != 2 || got[0] != "NEW" || got[1] != "RUNNING" {
			t.Fatalf("status params = %v, want [NEW RUNNING]", got)
		}
	})

	t.Run("statuses takes precedence over single status", func(t *testing.T) {
		t.Parallel()

		opts := ProcessListOptions{
			Status:   "COMPLETED",
			Statuses: []string{"NEW"},
		}
		values, _ := url.ParseQuery(strings.TrimPrefix(opts.QueryParams(), "?"))
		if got := values["status"]; len(got) != 1 || got[0] != "NEW" {
			t.Fatalf("status params = %v, want [NEW]", got)
		}
	})

	t.Run("single status still works", func(t *testing.T) {
		t.Parallel()

		opts := ProcessListOptions{Status: "RUNNING"}
		values, _ := url.ParseQuery(strings.TrimPrefix(opts.QueryParams(), "?"))
		if got := values.Get("status"); got != "RUNNING" {
			t.Fatalf("status = %q, want RUNNING", got)
		}
	})
}
