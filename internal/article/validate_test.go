package article

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLocalInputsFindsMissingMedia(t *testing.T) {
	dir := t.TempDir()
	doc := &Document{
		SourcePath: filepath.Join(dir, "article.md"),
		Title:      "Article",
		Blocks:     []Block{{Type: "paragraph", Text: "Body"}},
		Cover:      "./missing.png",
	}

	diagnostics := ValidateLocalInputs(doc)
	if !BlockingDiagnostics(diagnostics) {
		t.Fatalf("BlockingDiagnostics = false, want true: %#v", diagnostics)
	}
	if !hasValidationDiagnostic(diagnostics, "MEDIA_NOT_FOUND") {
		t.Fatalf("missing MEDIA_NOT_FOUND in %#v", diagnostics)
	}
}

func TestValidateLocalInputsRejectsUnsupportedImageExtension(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "cover.gif")
	if err := os.WriteFile(imagePath, []byte("gif"), 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	doc := &Document{
		SourcePath: filepath.Join(dir, "article.md"),
		Title:      "Article",
		Blocks:     []Block{{Type: "paragraph", Text: "Body"}},
		Cover:      "./cover.gif",
	}

	diagnostics := ValidateLocalInputs(doc)
	if !hasValidationDiagnostic(diagnostics, "UNSUPPORTED_IMAGE_TYPE") {
		t.Fatalf("missing UNSUPPORTED_IMAGE_TYPE in %#v", diagnostics)
	}
}

func TestValidateLocalInputsRejectsInvalidImageHeader(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "cover.png")
	if err := os.WriteFile(imagePath, []byte("not really png"), 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	doc := &Document{
		SourcePath: filepath.Join(dir, "article.md"),
		Title:      "Article",
		Blocks:     []Block{{Type: "paragraph", Text: "Body"}},
		Cover:      "./cover.png",
	}

	diagnostics := ValidateLocalInputs(doc)
	if !hasValidationDiagnostic(diagnostics, "INVALID_IMAGE_FILE") {
		t.Fatalf("missing INVALID_IMAGE_FILE in %#v", diagnostics)
	}
}

func TestValidateLocalInputsReadyDocumentHasNoDiagnostics(t *testing.T) {
	doc := &Document{
		SourcePath: "article.md",
		Title:      "Article",
		Blocks:     []Block{{Type: "paragraph", Text: "Body"}},
	}

	diagnostics := ValidateLocalInputs(doc)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if BlockingDiagnostics(diagnostics) {
		t.Fatalf("BlockingDiagnostics = true, want false")
	}
}

func hasValidationDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
