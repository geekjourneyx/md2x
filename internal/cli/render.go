package cli

import (
	"os"

	"github.com/geekjourneyx/md2x/internal/draftjs"
	"github.com/geekjourneyx/md2x/internal/markdown"
	"github.com/spf13/cobra"
)

type renderData struct {
	Title        string      `json:"title"`
	ContentState interface{} `json:"content_state"`
	Warnings     interface{} `json:"warnings"`
}

func newRenderCommand(opts *rootOptions) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "render <article.md>",
		Short: "Render a Markdown article",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "draftjs" {
				return &ExitError{Code: "UNSUPPORTED_FORMAT", Message: "render --format must be draftjs", Exit: 2}
			}

			sourcePath := args[0]
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				return &ExitError{Code: "INPUT_READ_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			doc, err := markdown.Parse(sourcePath, content)
			if err != nil {
				return &ExitError{Code: "MARKDOWN_PARSE_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			contentState, err := draftjs.Render(doc)
			if err != nil {
				return &ExitError{Code: "DRAFTJS_RENDER_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "rendered DraftJS content_state",
					Data: renderData{
						Title:        doc.Title,
						ContentState: contentState,
						Warnings:     doc.Warnings,
					},
				})
			}
			return writeJSONValue(cmd.OutOrStdout(), contentState)
		},
	}
	cmd.Flags().StringVar(&format, "format", "draftjs", "render output format")
	return cmd
}
