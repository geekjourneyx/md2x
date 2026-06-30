package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the md2x version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "md2x version",
					Data: map[string]string{
						"version": version,
					},
				})
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "md2x %s\n", version)
			return err
		},
	}
}
