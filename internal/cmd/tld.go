package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// TLDCmd is the parent command for TLD operations.
type TLDCmd struct {
	List TLDListCmd `cmd:"" help:"List supported TLDs"`
	Get  TLDGetCmd  `cmd:"" help:"Get TLD details"`
}

// TLDListCmd lists TLDs.
type TLDListCmd struct {
	Search string `help:"Only show TLDs or providers containing this text"`
	Limit  int    `help:"Max results" default:"50"`
	Offset int    `help:"Offset for pagination"`
}

// tldListRow is one line of `tld list`. Prices are major units and only set when a
// customer is configured.
type tldListRow struct {
	TLD           string  `json:"tld"`
	Provider      string  `json:"provider"`
	Currency      string  `json:"currency,omitempty"`
	CreatePrice   float64 `json:"createPrice,omitempty"`
	RenewPrice    float64 `json:"renewPrice,omitempty"`
	TransferPrice float64 `json:"transferPrice,omitempty"`
}

func (c *TLDListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	tlds, err := client.ListTLDs(ctx)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	page, total := filterTLDs(tlds, c.Search, c.Limit, c.Offset)
	pricelist := customerPricelist(ctx, client, flags.Verbose)

	rows := make([]tldListRow, 0, len(page))
	for _, t := range page {
		row := tldListRow{TLD: t.TLD, Provider: t.Provider}
		if pricelist != nil {
			row.fillPrices(pricelist)
		}
		rows = append(rows, row)
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"TLD", "PROVIDER"}
	if pricelist != nil {
		headers = append(headers, "CREATE", "RENEW", "TRANSFER", "CURRENCY")
	}
	table := make([][]string, 0, len(rows))
	for _, r := range rows {
		line := []string{r.TLD, r.Provider}
		if pricelist != nil {
			line = append(line, formatPrice(r.CreatePrice), formatPrice(r.RenewPrice), formatPrice(r.TransferPrice), r.Currency)
		}
		table = append(table, line)
	}

	if err := f.Output(rows, headers, table); err != nil {
		return err
	}
	warnIfCapped(total, len(rows), c.Limit)
	return nil
}

// filterTLDs applies --search (case-insensitive substring on TLD or provider) and then
// --offset/--limit. It returns the page and the number of matches before paging.
func filterTLDs(tlds []api.TLDEntry, search string, limit, offset int) (page []api.TLDEntry, total int) {
	search = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(search), "."))
	matches := make([]api.TLDEntry, 0, len(tlds))
	for _, t := range tlds {
		if search == "" || strings.Contains(strings.ToLower(t.TLD), search) || strings.Contains(strings.ToLower(t.Provider), search) {
			matches = append(matches, t)
		}
	}

	total = len(matches)
	if offset >= total {
		return []api.TLDEntry{}, total
	}
	matches = matches[max(offset, 0):]
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, total
}

// TLDGetCmd gets a single TLD.
type TLDGetCmd struct {
	TLD string `arg:"" help:"TLD name (e.g., com, net, io)"`
}

func (c *TLDGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	tld := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(c.TLD), "."))

	client := api.NewClient(apiKey)
	info, err := client.GetTLDInfo(ctx, tld)
	if err != nil {
		var notFound *api.NotFoundError
		if errors.As(err, &notFound) {
			err = fmt.Errorf("TLD %q not found; run 'rr tld list' to see supported TLDs", tld)
		}
		return &ExitError{Code: CodeAPI, Err: err}
	}

	m := info.Metadata
	idn := "no"
	if m.DomainSyntax.IDNSupport {
		idn = "yes"
	}

	kvPairs := [][2]string{
		{"TLD", tld},
		{"Provider", info.Provider},
		{"Create Periods", formatPeriods(m.CreateDomainPeriods)},
		{"Renew Periods", formatPeriods(m.RenewDomainPeriods)},
		{"Transfer Periods", formatPeriods(m.TransferDomainPeriods)},
		{"Auto-Renew Periods", formatPeriods(m.AutoRenewDomainPeriods)},
		{"Grace (days)", fmt.Sprintf("add %d, renew %d, auto-renew %d, transfer %d",
			m.AddGracePeriod, m.RenewGracePeriod, m.AutoRenewGracePeriod, m.TransferGracePeriod)},
		{"Redemption (days)", strconv.Itoa(m.RedemptionPeriod)},
		{"Name Length", fmt.Sprintf("%d-%d", m.DomainSyntax.MinLength, m.DomainSyntax.MaxLength)},
		{"IDN", idn},
		{"Premium", m.PremiumSupport},
		{"Features", strings.Join(m.FeaturesAvailable, ", ")},
	}

	if pricelist := customerPricelist(ctx, client, flags.Verbose); pricelist != nil {
		for _, action := range []string{"CREATE", "RENEW", "TRANSFER"} {
			if cents, cur, ok := pricelist.TLDActionPrice(tld, action); ok {
				label := strings.ToUpper(action[:1]) + strings.ToLower(action[1:]) + " Price"
				kvPairs = append(kvPairs, [2]string{label, fmt.Sprintf("%.2f %s", float64(cents)/100, cur)})
			}
		}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")
	return f.OutputSingle(info, kvPairs)
}

// customerPricelist fetches the configured customer's pricelist, or returns nil when no
// customer is configured or the fetch fails. Prices are optional extras, so failures only
// surface with --verbose.
func customerPricelist(ctx context.Context, client *api.Client, verbose bool) *api.Pricelist {
	customer, err := getCustomer()
	if err != nil {
		return nil
	}
	pricelist, err := client.GetPricelist(ctx, customer)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "debug: GetPricelist failed: %v\n", err)
		}
		return nil
	}
	return pricelist
}

// fillPrices sets the row's create/renew/transfer prices (major units) from the pricelist.
func (r *tldListRow) fillPrices(p *api.Pricelist) {
	for _, f := range []struct {
		action string
		dst    *float64
	}{
		{"CREATE", &r.CreatePrice},
		{"RENEW", &r.RenewPrice},
		{"TRANSFER", &r.TransferPrice},
	} {
		if cents, cur, ok := p.TLDActionPrice(r.TLD, f.action); ok {
			*f.dst = float64(cents) / 100
			if r.Currency == "" {
				r.Currency = cur
			}
		}
	}
}

func formatPrice(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", v)
}

// formatPeriods renders API periods (months) in years, collapsing a contiguous run of
// whole years into a range: [12 24 36] → "1-3 years", [12 60] → "1, 5 years".
func formatPeriods(months []int) string {
	if len(months) == 0 {
		return "-"
	}

	years := make([]int, 0, len(months))
	for _, m := range months {
		if m%12 != 0 {
			parts := make([]string, len(months))
			for i, mm := range months {
				parts[i] = strconv.Itoa(mm)
			}
			return strings.Join(parts, ", ") + " months"
		}
		years = append(years, m/12)
	}

	unit := " years"
	if len(years) == 1 && years[0] == 1 {
		unit = " year"
	}

	contiguous := len(years) > 2
	for i := 1; contiguous && i < len(years); i++ {
		contiguous = years[i] == years[i-1]+1
	}
	if contiguous {
		return fmt.Sprintf("%d-%d%s", years[0], years[len(years)-1], unit)
	}

	parts := make([]string, len(years))
	for i, y := range years {
		parts[i] = strconv.Itoa(y)
	}
	return strings.Join(parts, ", ") + unit
}
