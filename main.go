package main

import (
	"fmt"
	"os"

	"github.com/unkn0wn-root/resterm-runner/runner/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}

func run(args []string) error {
	return cli.Run(args, cli.Opt{
		Use:     "resterm-runner",
		Version: version,
		Commit:  commit,
		Date:    date,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
}
