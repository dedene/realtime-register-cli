package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/config"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// StatusCmd shows account status.
type StatusCmd struct{}

func (c *StatusCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}

	client := api.NewClient(apiKey)

	// Each metric is fetched independently so a single failing call degrades that
	// metric to "n/a" instead of failing the whole overview.
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	totalDomains := -1
	if resp, err := client.ListDomains(ctx, api.DomainListOptions{
		ListOptions: api.ListOptions{Limit: 1},
	}); err != nil {
		note(err)
	} else {
		totalDomains = resp.Pagination.Total
	}

	expiringDomains := -1
	if domains, _, err := client.ListExpiringDomains(ctx, 30, time.Now()); err != nil {
		note(err)
	} else {
		expiringDomains = len(domains)
	}

	pendingProcesses := -1
	if resp, err := client.ListProcesses(ctx, api.ProcessListOptions{
		ListOptions: api.ListOptions{Limit: 1},
		Statuses:    api.NonTerminalProcessStatuses,
	}); err != nil {
		note(err)
	} else {
		pendingProcesses = resp.Pagination.Total
	}

	status := map[string]any{
		"customer":         cfg.Customer,
		"totalDomains":     nullableInt(totalDomains),
		"expiringDomains":  nullableInt(expiringDomains),
		"pendingProcesses": nullableInt(pendingProcesses),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	// Warn on stderr regardless of output mode so it never pollutes --json stdout.
	if firstErr != nil {
		fmt.Fprintf(os.Stderr, "warning: some metrics unavailable: %v\n", firstErr)
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if flags.JSON {
		return f.Output(status, nil, nil)
	}

	kvPairs := [][2]string{
		{"Customer", cfg.Customer},
		{"Total Domains", metricValue(totalDomains)},
		{"Expiring (30d)", metricValue(expiringDomains)},
		{"Pending Processes", metricValue(pendingProcesses)},
	}

	return f.OutputSingle(status, kvPairs)
}

// metricValue renders a metric, showing "n/a" for the -1 sentinel.
func metricValue(n int) string {
	if n < 0 {
		return "n/a"
	}
	return strconv.Itoa(n)
}

// nullableInt returns nil for the -1 sentinel so JSON output emits null.
func nullableInt(n int) any {
	if n < 0 {
		return nil
	}
	return n
}
