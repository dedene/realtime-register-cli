package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestContactListOptionsQueryParams(t *testing.T) {
	t.Parallel()

	opts := ContactListOptions{ListOptions: ListOptions{Search: "acme", Limit: 10}}
	values, err := url.ParseQuery(strings.TrimPrefix(opts.QueryParams(), "?"))
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}
	if got := values.Get("q"); got != "acme" {
		t.Errorf("q = %q, want acme", got)
	}
	if values.Has("search") {
		t.Errorf("query unexpectedly contains 'search' (full: %q)", opts.QueryParams())
	}
}
