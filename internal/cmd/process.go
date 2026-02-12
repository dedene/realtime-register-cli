package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// ProcessCmd is the parent command for process operations.
type ProcessCmd struct {
	List   ProcessListCmd   `cmd:"" help:"List processes"`
	Get    ProcessGetCmd    `cmd:"" help:"Get process details"`
	Info   ProcessInfoCmd   `cmd:"" help:"Get extended process info"`
	Cancel ProcessCancelCmd `cmd:"" help:"Cancel a process"`
	Resend ProcessResendCmd `cmd:"" help:"Resend process notifications"`
}

// ProcessListCmd lists processes.
type ProcessListCmd struct {
	Status string `help:"Filter by status (pending, running, completed, failed)"`
	Limit  int    `help:"Max results" default:"50"`
	Offset int    `help:"Offset for pagination"`
}

func (c *ProcessListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)

	opts := api.ProcessListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
		},
		Status: c.Status,
	}

	resp, err := client.ListProcesses(ctx, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"ID", "STATUS", "ACTION", "ENTITY", "CREATED"}
	rows := make([][]string, 0, len(resp.Entities))
	for i := range resp.Entities {
		p := &resp.Entities[i]
		rows = append(rows, []string{
			fmt.Sprintf("%d", p.ID),
			p.Status,
			p.Action,
			p.Entity,
			p.CreatedDate.Format("2006-01-02 15:04"),
		})
	}

	return f.Output(resp.Entities, headers, rows)
}

// ProcessGetCmd gets a single process.
type ProcessGetCmd struct {
	ID int `arg:"" help:"Process ID"`
}

func (c *ProcessGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	process, err := client.GetProcess(ctx, c.ID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"ID", fmt.Sprintf("%d", process.ID)},
		{"Status", process.Status},
		{"Action", process.Action},
		{"Entity", process.Entity},
		{"Created", process.CreatedDate.Format("2006-01-02 15:04:05")},
	}

	return f.OutputSingle(process, kvPairs)
}

// ProcessInfoCmd gets extended process info.
type ProcessInfoCmd struct {
	ID int `arg:"" help:"Process ID"`
}

func (c *ProcessInfoCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	info, err := client.GetProcessInfo(ctx, c.ID)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if flags.JSON {
		return f.Output(info, nil, nil)
	}

	kvPairs := [][2]string{
		{"ID", fmt.Sprintf("%d", info.ID)},
		{"Status", info.Status},
		{"Action", info.Action},
		{"Entity", info.Entity},
		{"Created", info.CreatedDate.Format("2006-01-02 15:04:05")},
	}

	return f.OutputSingle(info, kvPairs)
}

// ProcessCancelCmd cancels a process.
type ProcessCancelCmd struct {
	ID int `arg:"" help:"Process ID to cancel"`
}

func (c *ProcessCancelCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Cancel process %d? [y/N]: ", c.ID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	if err := client.CancelProcess(ctx, c.ID); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Process %d cancelled.\n", c.ID)
	return nil
}

// ProcessResendCmd resends notifications.
type ProcessResendCmd struct {
	ID int `arg:"" help:"Process ID"`
}

func (c *ProcessResendCmd) Run(_ *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	if err := client.ResendProcess(ctx, c.ID); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Notifications resent for process %d.\n", c.ID)
	return nil
}
