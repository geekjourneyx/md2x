package mediafile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageMediaType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "cover.png", want: "image/png"},
		{path: "cover.jpg", want: "image/jpeg"},
		{path: "cover.jpeg", want: "image/jpeg"},
		{path: "cover.webp", want: "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ImageMediaType(tt.path)
			if err != nil {
				t.Fatalf("ImageMediaType returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ImageMediaType = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestImageMediaTypeRejectsUnsupportedExtension(t *testing.T) {
	if _, err := ImageMediaType("cover.gif"); err == nil {
		t.Fatal("ImageMediaType returned nil error, want unsupported extension")
	}
}

func TestSupportedImageExtension(t *testing.T) {
	if !SupportedImageExtension("photo.PNG") {
		t.Fatal("PNG extension should be supported case-insensitively")
	}
	if SupportedImageExtension("photo.gif") {
		t.Fatal("GIF extension should not be supported")
	}
}

func TestValidateImageFileAcceptsPNG(t *testing.T) {
	path := writeMediaTestFile(t, "cover.png", testPNG)

	mediaType, size, err := ValidateImageFile(path)
	if err != nil {
		t.Fatalf("ValidateImageFile returned error: %v", err)
	}
	if mediaType != "image/png" {
		t.Fatalf("mediaType = %q, want image/png", mediaType)
	}
	if size != int64(len(testPNG)) {
		t.Fatalf("size = %d, want %d", size, len(testPNG))
	}
}

func TestValidateImageFileRejectsInvalidHeader(t *testing.T) {
	path := writeMediaTestFile(t, "cover.png", []byte("not really png"))

	_, _, err := ValidateImageFile(path)
	if err == nil {
		t.Fatal("ValidateImageFile returned nil error, want invalid header")
	}
	if !strings.Contains(err.Error(), "content type") {
		t.Fatalf("error = %q, want content type context", err.Error())
	}
}

func TestValidateImageFileRejectsOversizedImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := file.Truncate(MaxImageUploadBytes + 1); err != nil {
		_ = file.Close()
		t.Fatalf("truncate image: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close image: %v", err)
	}

	_, _, err = ValidateImageFile(path)
	if err == nil {
		t.Fatal("ValidateImageFile returned nil error, want oversized error")
	}
	if !strings.Contains(err.Error(), "maximum supported size") {
		t.Fatalf("error = %q, want size context", err.Error())
	}
}

func writeMediaTestFile(t *testing.T, name string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	return path
}

var testPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
	0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
	0x44, 0xae, 0x42, 0x60, 0x82,
}
