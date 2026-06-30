package mediafile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MaxImageUploadBytes = 15 * 1024 * 1024

func ValidateImageFile(filePath string) (string, int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("stat image: %w", err)
	}
	if info.Size() > MaxImageUploadBytes {
		return "", 0, fmt.Errorf("image is %d bytes; maximum supported size is %d bytes", info.Size(), MaxImageUploadBytes)
	}

	mediaType, err := ImageMediaType(filePath)
	if err != nil {
		return "", 0, err
	}
	if err := ValidateImageHeader(filePath, mediaType); err != nil {
		return "", 0, err
	}
	return mediaType, info.Size(), nil
}

func SupportedImageExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func ImageMediaType(filePath string) (string, error) {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image type %q; supported extensions are .png, .jpg, .jpeg, .webp", filepath.Ext(filePath))
	}
}

func ValidateImageHeader(filePath, wantMediaType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open image: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var header [512]byte
	n, err := file.Read(header[:])
	if err != nil && err != io.EOF {
		return fmt.Errorf("read image header: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("image is empty")
	}

	got := http.DetectContentType(header[:n])
	if got != wantMediaType {
		return fmt.Errorf("image content type is %q, want %q", got, wantMediaType)
	}
	return nil
}
