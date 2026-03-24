package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dedene/realtime-register-cli/internal/api"
)

type fakeZoneClient struct {
	getZoneFn         func(context.Context, int) (*api.Zone, error)
	getZoneByDomainFn func(context.Context, string) (*api.Zone, error)
	listZonesFn       func(context.Context, api.ZoneListOptions) (*api.ListResponse[api.Zone], error)
}

func (f *fakeZoneClient) GetZone(ctx context.Context, id int) (*api.Zone, error) {
	return f.getZoneFn(ctx, id)
}

func (f *fakeZoneClient) GetZoneByDomain(ctx context.Context, domain string) (*api.Zone, error) {
	return f.getZoneByDomainFn(ctx, domain)
}

func (f *fakeZoneClient) ListZones(ctx context.Context, opts api.ZoneListOptions) (*api.ListResponse[api.Zone], error) {
	return f.listZonesFn(ctx, opts)
}

func TestResolveZoneBySelector_ZoneID(t *testing.T) {
	t.Parallel()

	client := &fakeZoneClient{
		getZoneFn: func(_ context.Context, id int) (*api.Zone, error) {
			return &api.Zone{ID: id, Name: "example.com"}, nil
		},
	}

	zone, err := resolveZone(context.Background(), client, 42, "")
	if err != nil {
		t.Fatalf("resolveZone() error = %v", err)
	}
	if zone.ID != 42 {
		t.Fatalf("resolveZone().ID = %d, want 42", zone.ID)
	}
}

func TestResolveZoneBySelector_DomainManagedLookup(t *testing.T) {
	t.Parallel()

	client := &fakeZoneClient{
		getZoneFn: func(_ context.Context, _ int) (*api.Zone, error) {
			t.Fatal("GetZone should not be called")
			return nil, nil
		},
		getZoneByDomainFn: func(_ context.Context, domain string) (*api.Zone, error) {
			if domain != "example.com" {
				t.Fatalf("domain = %q, want %q", domain, "example.com")
			}
			return &api.Zone{ID: 7, Name: domain}, nil
		},
		listZonesFn: func(_ context.Context, _ api.ZoneListOptions) (*api.ListResponse[api.Zone], error) {
			t.Fatal("ListZones should not be called after managed lookup succeeds")
			return nil, nil
		},
	}

	zone, err := resolveZone(context.Background(), client, 0, "example.com")
	if err != nil {
		t.Fatalf("resolveZone() error = %v", err)
	}
	if zone.ID != 7 {
		t.Fatalf("resolveZone().ID = %d, want 7", zone.ID)
	}
}

func TestResolveZoneBySelector_DomainFallbackToExactName(t *testing.T) {
	t.Parallel()

	client := &fakeZoneClient{
		getZoneByDomainFn: func(_ context.Context, _ string) (*api.Zone, error) {
			return nil, &api.NotFoundError{APIError: api.APIError{StatusCode: 404, Message: "not found"}}
		},
		listZonesFn: func(_ context.Context, opts api.ZoneListOptions) (*api.ListResponse[api.Zone], error) {
			if opts.Name != "example.com" {
				t.Fatalf("opts.Name = %q, want %q", opts.Name, "example.com")
			}
			if opts.Limit != 2 {
				t.Fatalf("opts.Limit = %d, want 2", opts.Limit)
			}
			return &api.ListResponse[api.Zone]{
				Entities: []api.Zone{{ID: 99, Name: "example.com"}},
			}, nil
		},
	}

	zone, err := resolveZone(context.Background(), client, 0, "example.com")
	if err != nil {
		t.Fatalf("resolveZone() error = %v", err)
	}
	if zone.ID != 99 {
		t.Fatalf("resolveZone().ID = %d, want 99", zone.ID)
	}
}

func TestResolveZoneBySelector_DomainLookupDoesNotFallbackOnServerError(t *testing.T) {
	t.Parallel()

	sentinel := &api.APIError{StatusCode: 500, Message: "boom"}
	client := &fakeZoneClient{
		getZoneByDomainFn: func(_ context.Context, _ string) (*api.Zone, error) {
			return nil, sentinel
		},
		listZonesFn: func(_ context.Context, _ api.ZoneListOptions) (*api.ListResponse[api.Zone], error) {
			t.Fatal("ListZones should not be called on non-not-found errors")
			return nil, nil
		},
	}

	_, err := resolveZone(context.Background(), client, 0, "example.com")
	if !errors.Is(err, sentinel) {
		t.Fatalf("resolveZone() error = %v, want %v", err, sentinel)
	}
}

func TestResolveZoneBySelector_DomainFallbackErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entities []api.Zone
		wantErr  string
	}{
		{
			name:    "no matches",
			wantErr: "zone not found for domain",
		},
		{
			name: "multiple matches",
			entities: []api.Zone{
				{ID: 1, Name: "example.com"},
				{ID: 2, Name: "example.com"},
			},
			wantErr: "multiple zones found for domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeZoneClient{
				getZoneByDomainFn: func(_ context.Context, _ string) (*api.Zone, error) {
					return nil, &api.NotFoundError{APIError: api.APIError{StatusCode: 404, Message: "not found"}}
				},
				listZonesFn: func(_ context.Context, _ api.ZoneListOptions) (*api.ListResponse[api.Zone], error) {
					return &api.ListResponse[api.Zone]{Entities: tt.entities}, nil
				},
			}

			_, err := resolveZone(context.Background(), client, 0, "example.com")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("resolveZone() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateZoneGetSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      int
		domain  string
		wantErr string
	}{
		{name: "id only", id: 42},
		{name: "domain only", domain: "example.com"},
		{name: "neither", wantErr: "provide either a zone ID or --domain"},
		{name: "both", id: 42, domain: "example.com", wantErr: "provide either a zone ID or --domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateZoneGetSelector(tt.id, tt.domain)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateZoneGetSelector() error = %v", err)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateZoneGetSelector() error = %v, want substring %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestZoneListCmdOptions(t *testing.T) {
	t.Parallel()

	cmd := ZoneListCmd{
		Search:  "example",
		Limit:   25,
		Offset:  5,
		Name:    "example.com",
		Managed: true,
		Service: "PREMIUM",
	}

	opts, err := cmd.options()
	if err != nil {
		t.Fatalf("options() error = %v", err)
	}
	if opts.Search != "example" || opts.Limit != 25 || opts.Offset != 5 {
		t.Fatalf("unexpected list options: %+v", opts.ListOptions)
	}
	if opts.Name != "example.com" {
		t.Fatalf("opts.Name = %q, want %q", opts.Name, "example.com")
	}
	if opts.Managed == nil || !*opts.Managed {
		t.Fatalf("opts.Managed = %v, want true", opts.Managed)
	}
	if opts.Service != "PREMIUM" {
		t.Fatalf("opts.Service = %q, want %q", opts.Service, "PREMIUM")
	}
}

func TestZoneListCmdOptions_NormalizesServiceCase(t *testing.T) {
	t.Parallel()

	opts, err := (ZoneListCmd{Service: "premium"}).options()
	if err != nil {
		t.Fatalf("options() error = %v", err)
	}
	if opts.Service != "PREMIUM" {
		t.Fatalf("opts.Service = %q, want %q", opts.Service, "PREMIUM")
	}
}

func TestZoneListCmdOptions_ManagedConflict(t *testing.T) {
	t.Parallel()

	_, err := (ZoneListCmd{Managed: true, Unmanaged: true}).options()
	if err == nil || !strings.Contains(err.Error(), "cannot use --managed and --unmanaged together") {
		t.Fatalf("options() error = %v", err)
	}
}

func TestParserAllowsZoneGetByDomain(t *testing.T) {
	t.Parallel()

	parser, err := newParser()
	if err != nil {
		t.Fatalf("newParser() error = %v", err)
	}

	if _, err := parser.Parse([]string{"zone", "get", "--domain", "example.com"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
}

func TestZoneListCmdOptions_RejectsInvalidService(t *testing.T) {
	t.Parallel()

	_, err := (ZoneListCmd{Service: "INVALID"}).options()
	if err == nil || !strings.Contains(err.Error(), "--service must be one of BASIC or PREMIUM") {
		t.Fatalf("options() error = %v, want service validation failure", err)
	}
}

func TestZoneLookupExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "api error",
			err:  &api.APIError{StatusCode: 500, Message: "boom"},
			want: CodeAPI,
		},
		{
			name: "not found api error",
			err:  &api.NotFoundError{APIError: api.APIError{StatusCode: 404, Message: "missing"}},
			want: CodeAPI,
		},
		{
			name: "local lookup error",
			err:  errors.New("multiple zones found for domain"),
			want: CodeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := zoneLookupExitCode(tt.err)
			if got != tt.want {
				t.Fatalf("zoneLookupExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestZoneCompletionIncludesCurrentSubcommands(t *testing.T) {
	t.Parallel()

	if !strings.Contains(bashCompletion, `zone)
            COMPREPLY=( $(compgen -W "list get create update delete sync record" -- ${cur}) )`) {
		t.Fatalf("bash zone completion missing updated subcommands")
	}
	if !strings.Contains(fishCompletion, `complete -c rr -n "__fish_seen_subcommand_from zone" -a "list get create update delete sync record"`) {
		t.Fatalf("fish zone completion missing updated subcommands")
	}
}
