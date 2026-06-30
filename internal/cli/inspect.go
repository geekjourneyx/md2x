package cli

import (
	"fmt"
	"os"

	"github.com/geekjourneyx/md2x/internal/article"
	"github.com/geekjourneyx/md2x/internal/markdown"
	"github.com/spf13/cobra"
)

type inspectData struct {
	Title      string               `json:"title"`
	Cover      string               `json:"cover"`
	BlockCount int                  `json:"block_count"`
	AssetCount int                  `json:"asset_count"`
	Warnings   []article.Diagnostic `json:"warnings"`
	Ready      bool                 `json:"ready"`
}

func newInspectCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <article.md>",
		Short: "Inspect a Markdown article",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath := args[0]
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				return &ExitError{Code: "INPUT_READ_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			doc, err := markdown.Parse(sourcePath, content)
			if err != nil {
				return &ExitError{Code: "MARKDOWN_PARSE_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			diagnostics := append(doc.Warnings, article.ValidateLocalInputs(doc)...)
			data := inspectData{
				Title:      doc.Title,
				Cover:      doc.Cover,
				BlockCount: len(doc.Blocks),
				AssetCount: len(doc.Assets),
				Warnings:   diagnostics,
				Ready:      !article.BlockingDiagnostics(diagnostics),
			}
			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "inspected Markdown article",
					Data:          data,
				})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), doc.Title)
			return err
		},
	}
}
