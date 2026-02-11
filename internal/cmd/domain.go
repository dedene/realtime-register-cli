package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/auth"
	"github.com/dedene/realtime-register-cli/internal/config"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// DomainCmd is the parent command for domain operations.
type DomainCmd struct {
	List           DomainListCmd           `cmd:"" help:"List domains"`
	Get            DomainGetCmd            `cmd:"" help:"Get domain details"`
	Check          DomainCheckCmd          `cmd:"" help:"Check domain availability"`
	CheckBulk      DomainCheckBulkCmd      `cmd:"" name:"check-bulk" help:"Bulk check availability (IsProxy)"`
	Register       DomainRegisterCmd       `cmd:"" help:"Register a domain"`
	Update         DomainUpdateCmd         `cmd:"" help:"Update domain settings"`
	Delete         DomainDeleteCmd         `cmd:"" help:"Delete a domain"`
	Renew          DomainRenewCmd          `cmd:"" help:"Renew a domain"`
	TransferIn     DomainTransferInCmd     `cmd:"" name:"transfer-in" help:"Transfer a domain in"`
	TransferStatus DomainTransferStatusCmd `cmd:"" name:"transfer-status" help:"Check transfer status"`
}

// DomainListCmd lists domains.
type DomainListCmd struct {
	Status         string `help:"Filter by status"`
	ExpiringWithin int    `help:"Show domains expiring within N days"`
	Search         string `help:"Search query"`
	Sort           string `help:"Sort by field (e.g., expiryDate, -expiryDate)" short:"s"`
	Limit          int    `help:"Max results" default:"50"`
	Offset         int    `help:"Offset for pagination"`
}

func (c *DomainListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	opts := api.DomainListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
			Search: c.Search,
		},
		Status:         c.Status,
		ExpiringWithin: c.ExpiringWithin,
		Order:          c.Sort,
	}

	resp, err := client.ListDomains(ctx, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"NAME", "STATUS", "EXPIRY", "AUTO-RENEW", "REGISTRANT"}
	var rows [][]string
	for _, d := range resp.Entities {
		autoRenew := "no"
		if d.AutoRenew {
			autoRenew = "yes"
		}
		rows = append(rows, []string{
			d.DomainName,
			strings.Join(d.Status, ", "),
			d.ExpiryDate.Format("2006-01-02"),
			autoRenew,
			d.Registrant,
		})
	}

	return f.Output(resp.Entities, headers, rows)
}

// DomainGetCmd gets a single domain.
type DomainGetCmd struct {
	Domain string `arg:"" help:"Domain name"`
}

func (c *DomainGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	domain, err := client.GetDomain(ctx, c.Domain)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	autoRenew := "no"
	if domain.AutoRenew {
		autoRenew = "yes"
	}

	kvPairs := [][2]string{
		{"Name", domain.DomainName},
		{"Status", strings.Join(domain.Status, ", ")},
		{"Expiry", domain.ExpiryDate.Format("2006-01-02")},
		{"Auto-Renew", autoRenew},
		{"Registrant", domain.Registrant},
	}

	return f.OutputSingle(domain, kvPairs)
}

// DomainCheckCmd checks domain availability.
type DomainCheckCmd struct {
	Domain string   `arg:"" help:"Domain name to check (use just name with --tlds)"`
	TLDs   []string `help:"Check multiple TLDs (provide name without TLD)" short:"t"`
}

func (c *DomainCheckCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	tlds := c.TLDs
	if len(tlds) == 0 {
		cfg, _ := config.ReadConfig()
		if cfg != nil && len(cfg.DefaultTLDs) > 0 {
			tlds = cfg.DefaultTLDs
		}
	}

	client := api.NewClient(apiKey)
	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if len(tlds) > 0 {
		name := c.Domain
		// Strip TLD only if domain contains a dot (user likely included TLD by mistake)
		if idx := strings.Index(name, "."); idx > 0 {
			name = name[:idx]
		}

		var results []api.DomainAvailability
		for _, tld := range tlds {
			domain := name + "." + strings.TrimPrefix(tld, ".")
			result, err := client.CheckDomain(ctx, domain)
			if err != nil {
				return &ExitError{Code: CodeAPI, Err: err}
			}
			results = append(results, *result)
		}

		headers := []string{"DOMAIN", "AVAILABLE", "PRICE"}
		var rows [][]string
		for _, r := range results {
			avail := "no"
			if r.Available {
				avail = "yes"
			}
			if r.Premium {
				avail += " (premium)"
			}
			price := ""
			if r.Price > 0 {
				price = fmt.Sprintf("%.2f", r.Price)
			}
			rows = append(rows, []string{r.Domain, avail, price})
		}
		return f.Output(results, headers, rows)
	}

	result, err := client.CheckDomain(ctx, c.Domain)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	// Fetch pricing if available and customer configured
	var currency string
	var noCustomer bool
	if result.Available && result.Price == 0 {
		cfg, _ := config.ReadConfig()
		if cfg == nil || cfg.Customer == "" {
			noCustomer = true
		} else {
			if idx := strings.LastIndex(c.Domain, "."); idx > 0 {
				tld := c.Domain[idx+1:]
				if pricelist, err := client.GetPricelist(ctx, cfg.Customer); err == nil {
					if cents, cur, ok := pricelist.GetTLDPrice(tld); ok {
						result.Price = float64(cents) / 100
						currency = cur
					} else if flags.Verbose {
						// Show first few products to debug naming
						fmt.Fprintf(os.Stderr, "debug: TLD %q not found in pricelist, sample products:\n", tld)
						for i, p := range pricelist.Prices {
							if i >= 5 {
								break
							}
							fmt.Fprintf(os.Stderr, "  - %s (%s)\n", p.Product, p.Action)
						}
					}
				} else if flags.Verbose {
					fmt.Fprintf(os.Stderr, "debug: GetPricelist failed: %v\n", err)
				}
			}
		}
	}

	available := "no"
	if result.Available {
		available = "yes"
	}
	premium := ""
	if result.Premium {
		premium = " (premium)"
	}

	kvPairs := [][2]string{
		{"Domain", result.Domain},
		{"Available", available + premium},
	}
	if result.Price > 0 {
		priceStr := fmt.Sprintf("%.2f", result.Price)
		if currency != "" {
			priceStr += " " + currency
		}
		priceStr += "/year"
		kvPairs = append(kvPairs, [2]string{"Price", priceStr})
	} else if noCustomer && result.Available {
		kvPairs = append(kvPairs, [2]string{"Price", "(set customer to show pricing)"})
	}

	return f.OutputSingle(result, kvPairs)
}

// DomainCheckBulkCmd checks multiple domains via IsProxy.
type DomainCheckBulkCmd struct {
	Domains []string `arg:"" help:"Domain names to check" required:""`
}

func (c *DomainCheckBulkCmd) Run(flags *RootFlags) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if len(c.Domains) > 50 {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("maximum 50 domains per request")}
	}

	client := api.NewIsProxyClient(apiKey)
	if err := client.Connect(); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}
	defer func() { _ = client.Close() }()

	results, err := client.CheckMany(c.Domains)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"DOMAIN", "AVAILABLE", "PRICE"}
	var rows [][]string
	for _, r := range results {
		avail := "no"
		if r.Available {
			avail = "yes"
		}
		price := ""
		if r.Price > 0 {
			price = fmt.Sprintf("%.2f", r.Price)
		}
		rows = append(rows, []string{
			r.Domain + "." + r.TLD,
			avail,
			price,
		})
	}

	return f.Output(results, headers, rows)
}

// DomainRegisterCmd registers a domain.
type DomainRegisterCmd struct {
	Domain     string   `arg:"" help:"Domain name to register"`
	Registrant string   `help:"Registrant contact handle" required:""`
	Period     int      `help:"Registration period in years" default:"1"`
	NS         []string `help:"Nameservers (comma-separated)"`
	AutoRenew  bool     `help:"Enable auto-renewal"`
	Privacy    bool     `help:"Enable privacy proxy"`
}

func (c *DomainRegisterCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Register %s for %d year(s)? [y/N]: ", c.Domain, c.Period)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	req := api.RegisterRequest{
		Period:       c.Period,
		Registrant:   c.Registrant,
		Nameservers:  c.NS,
		AutoRenew:    &c.AutoRenew,
		PrivacyProxy: &c.Privacy,
	}

	process, err := client.RegisterDomain(ctx, c.Domain, req)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"Process ID", fmt.Sprintf("%d", process.ID)},
		{"Status", process.Status},
		{"Domain", c.Domain},
	}

	return f.OutputSingle(process, kvPairs)
}

// DomainUpdateCmd updates domain settings.
type DomainUpdateCmd struct {
	Domain     string   `arg:"" help:"Domain name"`
	Registrant string   `help:"New registrant contact handle"`
	NS         []string `help:"New nameservers"`
	AutoRenew  *bool    `help:"Enable/disable auto-renewal"`
}

func (c *DomainUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	req := api.UpdateRequest{
		Registrant:  c.Registrant,
		Nameservers: c.NS,
		AutoRenew:   c.AutoRenew,
	}

	if err := client.UpdateDomain(ctx, c.Domain, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Domain %s updated.\n", c.Domain)
	return nil
}

// DomainDeleteCmd deletes a domain.
type DomainDeleteCmd struct {
	Domain string `arg:"" help:"Domain name to delete"`
}

func (c *DomainDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Delete domain %s? This cannot be undone. [y/N]: ", c.Domain)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	if err := client.DeleteDomain(ctx, c.Domain); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Domain %s deleted.\n", c.Domain)
	return nil
}

// DomainRenewCmd renews a domain.
type DomainRenewCmd struct {
	Domain string `arg:"" help:"Domain name to renew"`
	Period int    `help:"Renewal period in years" default:"1"`
}

func (c *DomainRenewCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Renew %s for %d year(s)? [y/N]: ", c.Domain, c.Period)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	process, err := client.RenewDomain(ctx, c.Domain, c.Period)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"Process ID", fmt.Sprintf("%d", process.ID)},
		{"Status", process.Status},
		{"Domain", c.Domain},
	}

	return f.OutputSingle(process, kvPairs)
}

// DomainTransferInCmd initiates a domain transfer.
type DomainTransferInCmd struct {
	Domain     string `arg:"" help:"Domain name to transfer"`
	AuthCode   string `help:"Authorization/EPP code" required:""`
	Registrant string `help:"Registrant contact handle"`
	AutoRenew  bool   `help:"Enable auto-renewal"`
}

func (c *DomainTransferInCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Transfer %s? [y/N]: ", c.Domain)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	req := api.TransferRequest{
		AuthCode:   c.AuthCode,
		Registrant: c.Registrant,
		AutoRenew:  &c.AutoRenew,
	}

	process, err := client.TransferDomain(ctx, c.Domain, req)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"Process ID", fmt.Sprintf("%d", process.ID)},
		{"Status", process.Status},
		{"Domain", c.Domain},
	}

	return f.OutputSingle(process, kvPairs)
}

// DomainTransferStatusCmd checks transfer status.
type DomainTransferStatusCmd struct {
	Domain string `arg:"" help:"Domain name"`
}

func (c *DomainTransferStatusCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	domain, err := client.GetDomain(ctx, c.Domain)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"Domain", domain.DomainName},
		{"Status", strings.Join(domain.Status, ", ")},
	}

	return f.OutputSingle(domain, kvPairs)
}

// getAPIKey retrieves the API key from env or keyring.
func getAPIKey() (string, error) {
	store, err := auth.NewStore("")
	if err != nil {
		return "", &ExitError{Code: CodeAuth, Err: fmt.Errorf("not authenticated: %w", err)}
	}
	key, err := store.GetAPIKey()
	if err != nil {
		return "", &ExitError{Code: CodeAuth, Err: fmt.Errorf("not authenticated: %w", err)}
	}
	return key, nil
}

// getCustomer retrieves customer from RR_CUSTOMER env or config.
func getCustomer() (string, error) {
	if customer := os.Getenv("RR_CUSTOMER"); customer != "" {
		return customer, nil
	}
	cfg, err := config.ReadConfig()
	if err != nil {
		return "", &ExitError{Code: CodeError, Err: fmt.Errorf("read config: %w", err)}
	}
	if cfg.Customer == "" {
		return "", &ExitError{Code: CodeError, Err: fmt.Errorf("customer not configured; run: rr config set customer <handle>")}
	}
	return cfg.Customer, nil
}
