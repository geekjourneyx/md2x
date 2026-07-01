package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.6"

type rootOptions struct {
	json       bool
	configPath string
}

func newRootCommand() (*cobra.Command, *rootOptions) {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:           "md2x",
		Short:         "Convert Markdown into publish-ready formats",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().BoolVar(&opts.json, "json", false, "write JSON output")
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "path to md2x config file")
	cmd.AddCommand(
		newVersionCommand(opts),
		newAuthCommand(opts),
		newConfigCommand(opts),
		newInspectCommand(opts),
		newRenderCommand(opts),
		newDraftCommand(opts),
	)
	return cmd, opts
}

func Execute() error {
	cmd, opts := newRootCommand()
	jsonRequested := jsonFlagRequested(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		exitErr, ok := err.(*ExitError)
		if !ok {
			exitErr = &ExitError{Code: "ERROR", Message: err.Error(), Exit: 1, Err: err}
		}
		if opts.json || jsonRequested {
			if err := writeJSON(os.Stdout, failureEnvelope(exitErr)); err != nil {
				fmt.Fprintln(os.Stderr, exitErr.Error())
			}
		} else {
			fmt.Fprintln(os.Stderr, exitErr.Error())
		}
		return exitErr
	}
	return nil
}

func jsonFlagRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		switch arg {
		case "--json", "--json=true":
			return true
		case "--json=false":
			return false
		}
	}
	return false
}
