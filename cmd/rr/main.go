package main

import (
	"os"
	_ "time/tzdata"

	"github.com/dedene/realtime-register-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(os.Args[1:]); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
