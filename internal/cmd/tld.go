package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// TLDCmd is the parent command for TLD operations.
type TLDCmd struct {
	List TLDListCmd `cmd:"" help:"List available TLDs"`
	Get  TLDGetCmd  `cmd:"" help:"Get TLD details"`
}

// TLDListCmd lists TLDs.
type TLDListCmd struct {
	Search string `help:"Search query"`
	Limit  int    `help:"Max results" default:"50"`
	Offset int    `help:"Offset for pagination"`
}

func (c *TLDListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	opts := api.TLDListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
			Search: c.Search,
		},
	}

	resp, err := client.ListTLDs(ctx, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"TLD", "CREATE PRICE", "RENEW PRICE", "TRANSFER PRICE"}
	var rows [][]string
	for _, t := range resp.Entities {
		rows = append(rows, []string{
			t.TLD,
			fmt.Sprintf("%.2f", t.PriceCreate),
			fmt.Sprintf("%.2f", t.PriceRenew),
			fmt.Sprintf("%.2f", t.PriceTransfer),
		})
	}

	return f.Output(resp.Entities, headers, rows)
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

	client := api.NewClient(apiKey)
	tld, err := client.GetTLD(ctx, c.TLD)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"TLD", tld.TLD},
		{"Create Price", fmt.Sprintf("%.2f", tld.PriceCreate)},
		{"Renew Price", fmt.Sprintf("%.2f", tld.PriceRenew)},
		{"Transfer Price", fmt.Sprintf("%.2f", tld.PriceTransfer)},
		{"Min Period", fmt.Sprintf("%d", tld.MinPeriod)},
		{"Max Period", fmt.Sprintf("%d", tld.MaxPeriod)},
	}

	return f.OutputSingle(tld, kvPairs)
}
