package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/dedene/realtime-register-cli/internal/errfmt"
)

// RootFlags are global flags available to all commands.
type RootFlags struct {
	JSON    bool   `help:"Output JSON to stdout" short:"j" env:"RR_JSON"`
	Plain   bool   `help:"Output plain TSV (for scripting)" env:"RR_PLAIN"`
	Verbose bool   `help:"HTTP debug logging" short:"v"`
	Yes     bool   `help:"Skip confirmation prompts" short:"y"`
	Color   string `help:"Color mode: auto|always|never" default:"auto" enum:"auto,always,never"`
}

// CLI is the top-level Kong CLI struct.
type CLI struct {
	RootFlags `embed:""`

	Version    kong.VersionFlag `help:"Print version and exit"`
	VersionCmd VersionCmd       `cmd:"" name:"version" help:"Show version information"`
	Domain     DomainCmd        `cmd:"" help:"Domain commands"`
	Process    ProcessCmd       `cmd:"" help:"Process commands"`
	Contact    ContactCmd       `cmd:"" help:"Contact commands"`
	Zone       ZoneCmd          `cmd:"" help:"DNS zone commands"`

	Auth       AuthCmd       `cmd:"" help:"Manage API key"`
	Config     ConfigCmd     `cmd:"" help:"Manage configuration"`
	Status     StatusCmd     `cmd:"" help:"Show account status"`
	TLD        TLDCmd        `cmd:"" name:"tld" help:"TLD commands"`
	Completion CompletionCmd `cmd:"" help:"Generate shell completions"`
}

type exitPanic struct{ code int }

// Execute runs the CLI with the given arguments.
func Execute(args []string) (err error) {
	parser, err := newParser()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				if ep.code == 0 {
					err = nil
					return
				}
				err = &ExitError{Code: ep.code, Err: errors.New("exited")}
				return
			}
			panic(r)
		}
	}()

	if len(args) == 0 {
		args = []string{"--help"}
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		parsedErr := wrapParseError(err)
		_, _ = fmt.Fprintln(os.Stderr, parsedErr)
		return parsedErr
	}

	err = kctx.Run()
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, errfmt.Format(err))
		return err
	}

	return nil
}

func wrapParseError(err error) error {
	if err == nil {
		return nil
	}
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return &ExitError{Code: CodeUsage, Err: parseErr}
	}
	return err
}

func newParser() (*kong.Kong, error) {
	cli := &CLI{}
	parser, err := kong.New(
		cli,
		kong.Name("rr"),
		kong.Description("RealtimeRegister CLI - Domain management from the command line"),
		kong.Vars{"version": VersionString()},
		kong.Writers(os.Stdout, os.Stderr),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
		kong.Help(helpPrinter),
		kong.ConfigureHelp(helpOptions()),
		kong.Bind(&cli.RootFlags),
	)
	if err != nil {
		return nil, err
	}

	return parser, nil
}
