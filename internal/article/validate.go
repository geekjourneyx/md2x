package article

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geekjourneyx/md2x/internal/mediafile"
)

func ValidateLocalInputs(doc *Document) []Diagnostic {
	if doc == nil {
		return []Diagnostic{{Severity: "error", Code: "NIL_DOCUMENT", Message: "document is nil"}}
	}

	var diagnostics []Diagnostic
	baseDir := filepath.Dir(doc.SourcePath)
	if doc.Title == "" {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "MISSING_TITLE", Message: "article title is empty"})
	}
	if len(doc.Blocks) == 0 {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "EMPTY_ARTICLE", Message: "article has no supported content blocks"})
	}
	if doc.Cover != "" {
		diagnostics = append(diagnostics, validateImagePath(baseDir, doc.Cover, "cover", 0)...)
	}
	for _, asset := range doc.Assets {
		if asset.Role != "body" {
			continue
		}
		diagnostics = append(diagnostics, validateImagePath(baseDir, asset.Source, "body image", asset.Index)...)
	}
	return diagnostics
}

func BlockingDiagnostics(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
		switch diagnostic.Code {
		case "UNSUPPORTED_MARKDOWN_BLOCK", "UNSUPPORTED_MIXED_IMAGE":
			return true
		}
	}
	return false
}

func validateImagePath(baseDir, source, role string, block int) []Diagnostic {
	path := source
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	if _, err := os.Stat(path); err != nil {
		return []Diagnostic{{
			Severity: "error",
			Code:     "MEDIA_NOT_FOUND",
			Message:  fmt.Sprintf("%s %q is not readable: %v", role, source, err),
			Block:    block,
		}}
	}
	if !mediafile.SupportedImageExtension(source) {
		return []Diagnostic{{
			Severity: "error",
			Code:     "UNSUPPORTED_IMAGE_TYPE",
			Message:  fmt.Sprintf("%s %q must be .png, .jpg, .jpeg, or .webp", role, source),
			Block:    block,
		}}
	}
	if _, _, err := mediafile.ValidateImageFile(path); err != nil {
		return []Diagnostic{{
			Severity: "error",
			Code:     "INVALID_IMAGE_FILE",
			Message:  fmt.Sprintf("%s %q is not a valid uploadable image: %v", role, source, err),
			Block:    block,
		}}
	}
	return nil
}
