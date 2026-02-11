package cmd

import (
	"fmt"
	"os"
	"strings"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

// VersionString returns the full version string.
func VersionString() string {
	var parts []string
	parts = append(parts, version)

	if commit != "" {
		parts = append(parts, "("+commit+")")
	}
	if date != "" {
		parts = append(parts, date)
	}

	return strings.Join(parts, " ")
}

// VersionCmd prints version information.
type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Fprintln(os.Stdout, "rr "+VersionString())
	return nil
}
