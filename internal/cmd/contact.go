package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/realtime-register-cli/internal/api"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// ContactCmd is the parent command for contact operations.
type ContactCmd struct {
	List   ContactListCmd   `cmd:"" help:"List contacts"`
	Get    ContactGetCmd    `cmd:"" help:"Get contact details"`
	Create ContactCreateCmd `cmd:"" help:"Create a contact"`
	Update ContactUpdateCmd `cmd:"" help:"Update a contact"`
	Delete ContactDeleteCmd `cmd:"" help:"Delete a contact"`
}

// ContactListCmd lists contacts.
type ContactListCmd struct {
	Search string `help:"Search query"`
	Limit  int    `help:"Max results" default:"50"`
	Offset int    `help:"Offset for pagination"`
}

func (c *ContactListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	customer, err := getCustomer()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	opts := api.ContactListOptions{
		ListOptions: api.ListOptions{
			Limit:  c.Limit,
			Offset: c.Offset,
			Search: c.Search,
		},
	}

	resp, err := client.ListContacts(ctx, customer, opts)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	headers := []string{"HANDLE", "NAME", "EMAIL", "PHONE", "COUNTRY"}
	var rows [][]string
	for _, ct := range resp.Entities {
		rows = append(rows, []string{
			ct.Handle,
			ct.Name,
			ct.Email,
			ct.Phone,
			ct.Country,
		})
	}

	return f.Output(resp.Entities, headers, rows)
}

// ContactGetCmd gets a single contact.
type ContactGetCmd struct {
	Handle string `arg:"" help:"Contact handle"`
}

func (c *ContactGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	customer, err := getCustomer()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	contact, err := client.GetContact(ctx, customer, c.Handle)
	if err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	kvPairs := [][2]string{
		{"Handle", contact.Handle},
		{"Name", contact.Name},
		{"Email", contact.Email},
		{"Phone", contact.Phone},
		{"Country", contact.Country},
	}

	return f.OutputSingle(contact, kvPairs)
}

// ContactCreateCmd creates a contact.
type ContactCreateCmd struct {
	Handle  string   `arg:"" help:"Contact handle (unique ID)"`
	Name    string   `help:"Full name" required:""`
	Email   string   `help:"Email address" required:""`
	Phone   string   `help:"Phone number (E.164 format)" required:""`
	Address []string `help:"Address lines" required:""`
	City    string   `help:"City" required:""`
	Postal  string   `help:"Postal code" required:""`
	Country string   `help:"Country (2-letter ISO code)" required:""`
	State   string   `help:"State/province"`
	Org     string   `help:"Organization name"`
}

func (c *ContactCreateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	customer, err := getCustomer()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	req := api.ContactRequest{
		Name:         c.Name,
		Organization: c.Org,
		Email:        c.Email,
		Phone:        c.Phone,
		Address:      c.Address,
		City:         c.City,
		State:        c.State,
		PostalCode:   c.Postal,
		Country:      c.Country,
	}

	if err := client.CreateContact(ctx, customer, c.Handle, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Contact %s created.\n", c.Handle)
	return nil
}

// ContactUpdateCmd updates a contact.
type ContactUpdateCmd struct {
	Handle  string   `arg:"" help:"Contact handle"`
	Name    string   `help:"Full name"`
	Email   string   `help:"Email address"`
	Phone   string   `help:"Phone number"`
	Address []string `help:"Address lines"`
	City    string   `help:"City"`
	Postal  string   `help:"Postal code"`
	Country string   `help:"Country"`
	State   string   `help:"State/province"`
	Org     string   `help:"Organization name"`
}

func (c *ContactUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	customer, err := getCustomer()
	if err != nil {
		return err
	}

	client := api.NewClient(apiKey)
	req := api.ContactRequest{
		Name:         c.Name,
		Organization: c.Org,
		Email:        c.Email,
		Phone:        c.Phone,
		Address:      c.Address,
		City:         c.City,
		State:        c.State,
		PostalCode:   c.Postal,
		Country:      c.Country,
	}

	if err := client.UpdateContact(ctx, customer, c.Handle, req); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Contact %s updated.\n", c.Handle)
	return nil
}

// ContactDeleteCmd deletes a contact.
type ContactDeleteCmd struct {
	Handle string `arg:"" help:"Contact handle to delete"`
}

func (c *ContactDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	customer, err := getCustomer()
	if err != nil {
		return err
	}

	if !flags.Yes {
		fmt.Printf("Delete contact %s? [y/N]: ", c.Handle)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := api.NewClient(apiKey)
	if err := client.DeleteContact(ctx, customer, c.Handle); err != nil {
		return &ExitError{Code: CodeAPI, Err: err}
	}

	fmt.Printf("Contact %s deleted.\n", c.Handle)
	return nil
}
