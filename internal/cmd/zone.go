package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// ZoneCmd is the parent command for zone operations.
type ZoneCmd struct {
	List   ZoneListCmd   `cmd:"" help:"List DNS zones"`
	Get    ZoneGetCmd    `cmd:"" help:"Get zone details"`
	Create ZoneCreateCmd `cmd:"" help:"Create a DNS zone"`
	Update ZoneUpdateCmd `cmd:"" help:"Update a DNS zone"`
	Delete ZoneDeleteCmd `cmd:"" help:"Delete a DNS zone"`
	Sync   ZoneSyncCmd   `cmd:"" help:"Sync zone from YAML file"`
	Record ZoneRecordCmd `cmd:"" help:"Manage DNS records"`
}

// ZoneListCmd lists zones.
type ZoneListCmd struct {
	Search    string `help:"Search query"`
	Name      string `help:"Filter by exact zone name"`
	Managed   bool   `help:"Only show managed zones"`
	Unmanaged bool   `help:"Only show unmanaged zones"`
	Service   string `help:"Filter by DNS service (BASIC or PREMIUM)"`
	Limit     int    `help:"Max results" default:"50"`
	Offset    int    `help:"Offset for pagination"`
}

func (c *ZoneListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	opts, err := c.options()
	if err != nil {
		return &ExitError{Code: CodeUsage, Err: err}
	}

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	resp, err := client.ListZones(ctx, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"ID", "NAME", "RECORDS"}
	rows := make([][]string, 0, len(resp.Entities))
	for i := range resp.Entities {
		z := &resp.Entities[i]
		rows = append(rows, []string{
			fmt.Sprintf("%d", z.ID),
			z.Name,
			fmt.Sprintf("%d", len(z.Records)),
		})
	}

	if err := f.Output(resp.Entities, headers, rows); err != nil {
		return err
	}
	warnIfCapped(resp.Pagination.Total, len(resp.Entities), c.Limit)
	return nil
}

func (c ZoneListCmd) options() (api.ZoneListOptions, error) {
	if c.Managed && c.Unmanaged {
		return api.ZoneListOptions{}, fmt.Errorf("cannot use --managed and --unmanaged together")
	}
	service := strings.ToUpper(strings.TrimSpace(c.Service))
	if service != "" && service != "BASIC" && service != "PREMIUM" {
		return api.ZoneListOptions{}, fmt.Errorf("--service must be one of BASIC or PREMIUM")
	}

	opts := api.ZoneListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
			Search: c.Search,
		},
		Name:    c.Name,
		Service: service,
	}

	switch {
	case c.Managed:
		opts.Managed = boolPtr(true)
	case c.Unmanaged:
		opts.Managed = boolPtr(false)
	}

	return opts, nil
}

// ZoneGetCmd gets a single zone.
type ZoneGetCmd struct {
	Zone   string `arg:"" optional:"" name:"zone" help:"Zone ID or domain name"`
	Domain string `help:"Domain name to fetch zone for"`
}

func (c *ZoneGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	id, domain, err := zoneGetSelector(c.Zone, c.Domain)
	if err != nil {
		return &ExitError{Code: CodeUsage, Err: err}
	}

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if flags.JSON {
		return f.Output(zone, nil, nil)
	}

	kvPairs := [][2]string{
		{"ID", fmt.Sprintf("%d", zone.ID)},
		{"Name", zone.Name},
		{"Records", fmt.Sprintf("%d", len(zone.Records))},
	}

	if err := f.OutputSingle(zone, kvPairs); err != nil {
		return err
	}

	if len(zone.Records) > 0 {
		fmt.Println()
		fmt.Println("Records:")
		headers := []string{"TYPE", "NAME", "CONTENT", "TTL"}
		var rows [][]string
		for _, r := range zone.Records {
			rows = append(rows, []string{
				r.Type,
				r.Name,
				r.Content,
				fmt.Sprintf("%d", r.TTL),
			})
		}
		return output.RenderTable(os.Stdout, headers, rows, f.Colors)
	}

	return nil
}

type zoneLookupClient interface {
	GetZone(context.Context, int) (*api.Zone, error)
	GetZoneByDomain(context.Context, string) (*api.Zone, error)
	ListZones(context.Context, api.ZoneListOptions) (*api.ListResponse[api.Zone], error)
}

// zoneGetSelector derives the (id, domain) lookup pair from the positional argument
// (a numeric zone ID or a domain name) and the optional --domain flag.
func zoneGetSelector(arg, domainFlag string) (id int, domain string, err error) {
	arg = strings.TrimSpace(arg)
	switch {
	case arg != "" && domainFlag != "":
		return 0, "", fmt.Errorf("provide either a zone ID/domain argument or --domain, not both")
	case domainFlag != "":
		return 0, domainFlag, nil
	case arg != "":
		id, domain = parseZoneArg(arg)
		return id, domain, nil
	default:
		return 0, "", fmt.Errorf("provide a zone ID or domain name")
	}
}

// parseZoneArg interprets a zone argument as either a numeric zone ID or a domain name.
func parseZoneArg(arg string) (id int, domain string) {
	trimmed := strings.TrimSpace(arg)
	if n, err := strconv.Atoi(trimmed); err == nil && n > 0 {
		return n, ""
	}
	return 0, trimmed
}

// resolveZoneID resolves a zone argument (ID or domain) to a numeric zone ID.
func resolveZoneID(ctx context.Context, client zoneLookupClient, arg string) (int, error) {
	id, domain := parseZoneArg(arg)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return 0, err
	}
	return zone.ID, nil
}

func resolveZone(ctx context.Context, client zoneLookupClient, id int, domain string) (*api.Zone, error) {
	if id > 0 {
		return client.GetZone(ctx, id)
	}

	zone, err := client.GetZoneByDomain(ctx, domain)
	if err == nil {
		return zone, nil
	}

	var notFoundErr *api.NotFoundError
	if !errors.As(err, &notFoundErr) {
		return nil, err
	}

	resp, err := client.ListZones(ctx, api.ZoneListOptions{
		ListOptions: api.ListOptions{Limit: 2},
		Name:        domain,
	})
	if err != nil {
		return nil, err
	}

	switch len(resp.Entities) {
	case 0:
		return nil, fmt.Errorf("zone not found for domain %q", domain)
	case 1:
		// Fetch by ID so record-mutating callers always get the full record set,
		// even if the list endpoint were ever to return zones without records.
		return client.GetZone(ctx, resp.Entities[0].ID)
	default:
		return nil, fmt.Errorf("multiple zones found for domain %q; use rr zone list --name %s or a zone ID", domain, domain)
	}
}

func zoneLookupExitCode(err error) int {
	switch {
	case err == nil:
		return CodeSuccess
	case isAPIError(err):
		return CodeAPI
	default:
		return CodeError
	}
}

func isAPIError(err error) bool {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return true
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return true
	}

	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	var notFoundErr *api.NotFoundError
	return errors.As(err, &notFoundErr)
}

func boolPtr(v bool) *bool {
	return &v
}

// ZoneCreateCmd creates a zone.
type ZoneCreateCmd struct {
	Name string `arg:"" help:"Zone name (domain)"`
	TTL  int    `help:"Default TTL" default:"3600"`
}

func (c *ZoneCreateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	req := api.ZoneRequest{
		Name: c.Name,
		TTL:  c.TTL,
	}

	id, err := client.CreateZone(ctx, &req)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if flags.JSON {
		return f.Output(map[string]any{"id": id, "name": c.Name}, nil, nil)
	}

	fmt.Printf("Zone created with ID %d.\n", id)
	return nil
}

// ZoneUpdateCmd updates a zone.
type ZoneUpdateCmd struct {
	Zone string `arg:"" name:"zone" help:"Zone ID or domain name"`
	TTL  int    `help:"Default TTL"`
}

func (c *ZoneUpdateCmd) Run(_ *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	id, err := resolveZoneID(ctx, client, c.Zone)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	req := api.ZoneRequest{
		TTL: c.TTL,
	}

	if err := client.UpdateZone(ctx, id, &req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d updated.\n", id)
	return nil
}

// ZoneDeleteCmd deletes a zone.
type ZoneDeleteCmd struct {
	Zone string `arg:"" name:"zone" help:"Zone ID or domain name to delete"`
}

func (c *ZoneDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	id, domain := parseZoneArg(c.Zone)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	if !flags.Yes {
		fmt.Printf("Delete zone %d (%s)? This cannot be undone. [y/N]: ", zone.ID, zone.Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	if err := client.DeleteZone(ctx, zone.ID); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d deleted.\n", zone.ID)
	return nil
}

// ZoneRecordCmd is the parent for record subcommands.
type ZoneRecordCmd struct {
	Add    ZoneRecordAddCmd    `cmd:"" help:"Add a DNS record"`
	Update ZoneRecordUpdateCmd `cmd:"" help:"Update a DNS record"`
	Delete ZoneRecordDeleteCmd `cmd:"" help:"Delete a DNS record"`
}

// ZoneRecordAddCmd adds a record to a zone.
type ZoneRecordAddCmd struct {
	Zone     string `arg:"" name:"zone" help:"Zone ID or domain name"`
	Type     string `help:"Record type (A, AAAA, CNAME, MX, TXT, etc.)" required:""`
	Name     string `help:"Record name (@ for apex)" required:""`
	Content  string `help:"Record content" required:""`
	TTL      int    `help:"TTL in seconds" default:"3600"`
	Priority int    `help:"Priority (for MX/SRV)" default:"0"`
}

func (c *ZoneRecordAddCmd) Run(_ *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	id, domain := parseZoneArg(c.Zone)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	newRecord := api.DNSRecord{
		Name:    apexRecordName(c.Name, zone.Name),
		Type:    strings.ToUpper(c.Type),
		Content: c.Content,
		TTL:     c.TTL,
		Prio:    c.Priority,
	}
	zone.Records = append(zone.Records, newRecord)

	req := api.ZoneRequest{Records: zone.Records}
	if err := client.UpdateZone(ctx, zone.ID, &req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Record %s %s added to zone %d (%s).\n", c.Type, newRecord.Name, zone.ID, zone.Name)
	return nil
}

// ZoneRecordUpdateCmd updates a record in a zone.
type ZoneRecordUpdateCmd struct {
	Zone       string `arg:"" name:"zone" help:"Zone ID or domain name"`
	Type       string `help:"Record type (A, AAAA, CNAME, etc.)" required:""`
	Name       string `help:"Record name (@ for apex)" required:""`
	Content    string `help:"New content" required:""`
	OldContent string `help:"Old content (for disambiguation when multiple records match)"`
	TTL        int    `help:"New TTL" default:"3600"`
	Priority   int    `help:"New priority (for MX/SRV)" default:"-1"`
}

func (c *ZoneRecordUpdateCmd) Run(_ *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	id, domain := parseZoneArg(c.Zone)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	if len(zone.Records) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("zone has no records")}
	}

	typ := strings.ToUpper(c.Type)
	name := apexRecordName(c.Name, zone.Name)
	indices := findRecords(zone.Records, typ, name, c.OldContent)

	if len(indices) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("no %s record found for name %q", typ, name)}
	}
	if len(indices) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple %s records found for %q:\n", typ, name)
		for _, i := range indices {
			r := zone.Records[i]
			fmt.Fprintf(os.Stderr, "  - %s\n", r.Content)
		}
		return &ExitError{Code: CodeError, Err: fmt.Errorf("use --old-content to specify which record to update")}
	}

	idx := indices[0]
	old := zone.Records[idx]
	zone.Records[idx].Content = c.Content
	if c.TTL > 0 {
		zone.Records[idx].TTL = c.TTL
	}
	if c.Priority >= 0 {
		zone.Records[idx].Prio = c.Priority
	}

	req := api.ZoneRequest{Records: zone.Records}
	if err := client.UpdateZone(ctx, zone.ID, &req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Updated %s %s: %s → %s\n", typ, name, old.Content, c.Content)
	return nil
}

// ZoneRecordDeleteCmd deletes a record from a zone.
type ZoneRecordDeleteCmd struct {
	Zone    string `arg:"" name:"zone" help:"Zone ID or domain name"`
	Type    string `help:"Record type (A, AAAA, CNAME, etc.)" required:""`
	Name    string `help:"Record name (@ for apex)" required:""`
	Content string `help:"Record content (for disambiguation when multiple records match)"`
}

func (c *ZoneRecordDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	id, domain := parseZoneArg(c.Zone)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	if len(zone.Records) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("zone has no records")}
	}

	typ := strings.ToUpper(c.Type)
	name := apexRecordName(c.Name, zone.Name)
	indices := findRecords(zone.Records, typ, name, c.Content)

	if len(indices) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("no %s record found for name %q", typ, name)}
	}
	if len(indices) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple %s records found for %q:\n", typ, name)
		for _, i := range indices {
			r := zone.Records[i]
			fmt.Fprintf(os.Stderr, "  - %s\n", r.Content)
		}
		return &ExitError{Code: CodeError, Err: fmt.Errorf("use --content to specify which record to delete")}
	}

	idx := indices[0]
	record := zone.Records[idx]

	if !flags.Yes {
		fmt.Printf("Delete %s %s → %s? [y/N]: ", typ, name, record.Content)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	zone.Records = append(zone.Records[:idx], zone.Records[idx+1:]...)

	req := api.ZoneRequest{Records: zone.Records}
	if err := client.UpdateZone(ctx, zone.ID, &req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Deleted %s %s → %s\n", typ, name, record.Content)
	return nil
}

// apexRecordName resolves the conventional "@" apex marker to the zone name. RR stores
// record names fully-qualified, so the apex record is named after the zone itself.
func apexRecordName(name, zoneName string) string {
	if name == "@" || name == "" {
		return zoneName
	}
	return name
}

// findRecords returns indices of records matching type, name, and optionally content.
func findRecords(records []api.DNSRecord, typ, name, content string) []int {
	var indices []int
	for i, r := range records {
		if strings.EqualFold(r.Type, typ) && r.Name == name {
			if content == "" || r.Content == content {
				indices = append(indices, i)
			}
		}
	}
	return indices
}

// ZoneSyncCmd syncs zone records from a YAML file.
type ZoneSyncCmd struct {
	Zone string `arg:"" name:"zone" help:"Zone ID or domain name"`
	File string `help:"YAML file with records" required:"" type:"existingfile"`
}

// ZoneSyncFile represents the YAML structure for zone sync.
type ZoneSyncFile struct {
	Records []ZoneSyncRecord `yaml:"records"`
}

// ZoneSyncRecord is a record definition in the sync file.
type ZoneSyncRecord struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Content  string `yaml:"content"`
	TTL      int    `yaml:"ttl"`
	Priority int    `yaml:"priority"`
}

func (c *ZoneSyncCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(c.File)
	if err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("read file: %w", err)}
	}

	var syncFile ZoneSyncFile
	if err := yaml.Unmarshal(data, &syncFile); err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("parse YAML: %w", err)}
	}

	client := api.NewClient(apiKey)

	id, domain := parseZoneArg(c.Zone)
	zone, err := resolveZone(ctx, client, id, domain)
	if err != nil {
		return &ExitError{Code: zoneLookupExitCode(err), Err: err}
	}

	newRecords := make([]api.DNSRecord, 0, len(syncFile.Records))
	for _, r := range syncFile.Records {
		ttl := r.TTL
		if ttl == 0 {
			ttl = 3600
		}
		newRecords = append(newRecords, api.DNSRecord{
			Name:    r.Name,
			Type:    strings.ToUpper(r.Type),
			Content: r.Content,
			TTL:     ttl,
			Prio:    r.Priority,
		})
	}

	fmt.Printf("Zone %d (%s): %d current records → %d new records\n",
		zone.ID, zone.Name, len(zone.Records), len(newRecords))

	if !flags.Yes {
		fmt.Printf("Apply changes? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	req := api.ZoneRequest{Records: newRecords}
	if err := client.UpdateZone(ctx, zone.ID, &req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d synced with %d records.\n", zone.ID, len(newRecords))
	return nil
}
