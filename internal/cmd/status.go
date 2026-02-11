package cmd

import (
	"context"
	"fmt"
	"os"
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

	domains, err := client.ListDomains(ctx, api.DomainListOptions{
		ListOptions: api.ListOptions{Limit: 1},
	})
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	expiring, err := client.ListDomains(ctx, api.DomainListOptions{
		ListOptions:    api.ListOptions{Limit: 1},
		ExpiringWithin: 30,
	})
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	processes, err := client.ListProcesses(ctx, api.ProcessListOptions{
		ListOptions: api.ListOptions{Limit: 1},
		Status:      "pending",
	})
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	status := map[string]any{
		"customer":         cfg.Customer,
		"totalDomains":     domains.Pagination.Total,
		"expiringDomains":  expiring.Pagination.Total,
		"pendingProcesses": processes.Pagination.Total,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	if flags.JSON {
		return f.Output(status, nil, nil)
	}

	kvPairs := [][2]string{
		{"Customer", cfg.Customer},
		{"Total Domains", fmt.Sprintf("%d", domains.Pagination.Total)},
		{"Expiring (30d)", fmt.Sprintf("%d", expiring.Pagination.Total)},
		{"Pending Processes", fmt.Sprintf("%d", processes.Pagination.Total)},
	}

	return f.OutputSingle(status, kvPairs)
}
