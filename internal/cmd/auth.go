package cmd

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/dedene/realtime-register-cli/internal/auth"
)

// AuthCmd manages API key.
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Store API key"`
	Status AuthStatusCmd `cmd:"" help:"Show authentication status"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove API key"`
}

// AuthLoginCmd stores API key.
type AuthLoginCmd struct{}

func (c *AuthLoginCmd) Run(flags *RootFlags) error {
	fmt.Print("Enter API key: ")
	keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("read key: %w", err)}
	}

	key := string(keyBytes)
	if key == "" {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("API key cannot be empty")}
	}

	store, err := auth.NewStore("")
	if err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("open keyring: %w", err)}
	}

	if err := store.SetAPIKey(key); err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("store key: %w", err)}
	}

	fmt.Println("API key stored.")
	return nil
}

// AuthStatusCmd shows auth status.
type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(flags *RootFlags) error {
	if os.Getenv("RR_API_KEY") != "" {
		fmt.Println("Authenticated via RR_API_KEY environment variable")
		return nil
	}

	store, err := auth.NewStore("")
	if err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("open keyring: %w", err)}
	}

	if store.HasAPIKey() {
		fmt.Println("Authenticated via keyring")
	} else {
		fmt.Println("Not authenticated")
		fmt.Println("\nTo authenticate:")
		fmt.Println("  rr auth login")
		fmt.Println("\nOr set environment variable:")
		fmt.Println("  export RR_API_KEY=your-api-key")
	}

	return nil
}

// AuthLogoutCmd removes API key.
type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(flags *RootFlags) error {
	store, err := auth.NewStore("")
	if err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("open keyring: %w", err)}
	}

	if err := store.DeleteAPIKey(); err != nil {
		return &ExitError{Code: CodeError, Err: fmt.Errorf("delete key: %w", err)}
	}

	fmt.Println("API key removed.")
	return nil
}
