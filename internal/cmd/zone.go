package cmd

import (
	"context"
	"fmt"
	"os"
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
	Search string `help:"Search query"`
	Limit  int    `help:"Max results" default:"50"`
	Offset int    `help:"Offset for pagination"`
}

func (c *ZoneListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	opts := api.ZoneListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
			Search: c.Search,
		},
	}

	resp, err := client.ListZones(ctx, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"ID", "NAME", "RECORDS"}
	var rows [][]string
	for _, z := range resp.Entities {
		rows = append(rows, []string{
			fmt.Sprintf("%d", z.ID),
			z.Name,
			fmt.Sprintf("%d", len(z.Records)),
		})
	}

	return f.Output(resp.Entities, headers, rows)
}

// ZoneGetCmd gets a single zone.
type ZoneGetCmd struct {
	ID int `arg:"" help:"Zone ID"`
}

func (c *ZoneGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	zone, err := client.GetZone(ctx, c.ID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
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

	id, err := client.CreateZone(ctx, req)
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
	ID  int `arg:"" help:"Zone ID"`
	TTL int `help:"Default TTL"`
}

func (c *ZoneUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	req := api.ZoneRequest{
		TTL: c.TTL,
	}

	if err := client.UpdateZone(ctx, c.ID, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d updated.\n", c.ID)
	return nil
}

// ZoneDeleteCmd deletes a zone.
type ZoneDeleteCmd struct {
	ID int `arg:"" help:"Zone ID to delete"`
}

func (c *ZoneDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Delete zone %d? This cannot be undone. [y/N]: ", c.ID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	if err := client.DeleteZone(ctx, c.ID); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d deleted.\n", c.ID)
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
	ZoneID   int    `arg:"" help:"Zone ID"`
	Type     string `help:"Record type (A, AAAA, CNAME, MX, TXT, etc.)" required:""`
	Name     string `help:"Record name (@ for apex)" required:""`
	Content  string `help:"Record content" required:""`
	TTL      int    `help:"TTL in seconds" default:"3600"`
	Priority int    `help:"Priority (for MX/SRV)" default:"0"`
}

func (c *ZoneRecordAddCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	zone, err := client.GetZone(ctx, c.ZoneID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	newRecord := api.DNSRecord{
		Name:    c.Name,
		Type:    strings.ToUpper(c.Type),
		Content: c.Content,
		TTL:     c.TTL,
		Prio:    c.Priority,
	}
	zone.Records = append(zone.Records, newRecord)

	req := api.ZoneRequest{Records: zone.Records}
	if err := client.UpdateZone(ctx, c.ZoneID, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Record %s %s added to zone %d.\n", c.Type, c.Name, c.ZoneID)
	return nil
}

// ZoneRecordUpdateCmd updates a record in a zone.
type ZoneRecordUpdateCmd struct {
	ZoneID     int    `arg:"" help:"Zone ID"`
	Type       string `help:"Record type (A, AAAA, CNAME, etc.)" required:""`
	Name       string `help:"Record name (@ for apex)" required:""`
	Content    string `help:"New content" required:""`
	OldContent string `help:"Old content (for disambiguation when multiple records match)"`
	TTL        int    `help:"New TTL" default:"3600"`
	Priority   int    `help:"New priority (for MX/SRV)" default:"-1"`
}

func (c *ZoneRecordUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	zone, err := client.GetZone(ctx, c.ZoneID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	if len(zone.Records) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("zone has no records")}
	}

	typ := strings.ToUpper(c.Type)
	indices := findRecords(zone.Records, typ, c.Name, c.OldContent)

	if len(indices) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("no %s record found for name %q", typ, c.Name)}
	}
	if len(indices) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple %s records found for %q:\n", typ, c.Name)
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
	if err := client.UpdateZone(ctx, c.ZoneID, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Updated %s %s: %s → %s\n", typ, c.Name, old.Content, c.Content)
	return nil
}

// ZoneRecordDeleteCmd deletes a record from a zone.
type ZoneRecordDeleteCmd struct {
	ZoneID  int    `arg:"" help:"Zone ID"`
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

	zone, err := client.GetZone(ctx, c.ZoneID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	if len(zone.Records) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("zone has no records")}
	}

	typ := strings.ToUpper(c.Type)
	indices := findRecords(zone.Records, typ, c.Name, c.Content)

	if len(indices) == 0 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("no %s record found for name %q", typ, c.Name)}
	}
	if len(indices) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple %s records found for %q:\n", typ, c.Name)
		for _, i := range indices {
			r := zone.Records[i]
			fmt.Fprintf(os.Stderr, "  - %s\n", r.Content)
		}
		return &ExitError{Code: CodeError, Err: fmt.Errorf("use --content to specify which record to delete")}
	}

	idx := indices[0]
	record := zone.Records[idx]

	if !flags.Yes {
		fmt.Printf("Delete %s %s → %s? [y/N]: ", typ, c.Name, record.Content)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	zone.Records = append(zone.Records[:idx], zone.Records[idx+1:]...)

	req := api.ZoneRequest{Records: zone.Records}
	if err := client.UpdateZone(ctx, c.ZoneID, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Deleted %s %s → %s\n", typ, c.Name, record.Content)
	return nil
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
	ZoneID int    `arg:"" help:"Zone ID"`
	File   string `help:"YAML file with records" required:"" type:"existingfile"`
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

	zone, err := client.GetZone(ctx, c.ZoneID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	var newRecords []api.DNSRecord
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
	if err := client.UpdateZone(ctx, c.ZoneID, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Zone %d synced with %d records.\n", c.ZoneID, len(newRecords))
	return nil
}
