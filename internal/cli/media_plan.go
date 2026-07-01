package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/geekjourneyx/md2x/internal/article"
	"github.com/geekjourneyx/md2x/internal/mediafile"
	"github.com/geekjourneyx/md2x/internal/xapi"
)

type xRequestEstimate struct {
	MediaUpload int `json:"media_upload"`
	CreateDraft int `json:"create_draft"`
	Total       int `json:"total"`
}

type mediaUploadCache struct {
	byFingerprint map[string]*xapi.UploadMediaResult
}

func newMediaUploadCache() *mediaUploadCache {
	return &mediaUploadCache{byFingerprint: map[string]*xapi.UploadMediaResult{}}
}

func (cache *mediaUploadCache) uploadImage(client *xapi.Client, path string) (*xapi.UploadMediaResult, error) {
	fingerprint, err := mediaFingerprint(path)
	if err != nil {
		return nil, &xapi.MediaValidationError{Path: path, Err: err}
	}
	if result, ok := cache.byFingerprint[fingerprint]; ok {
		return result, nil
	}
	result, err := client.UploadImage(path)
	if err != nil {
		return nil, err
	}
	cache.byFingerprint[fingerprint] = result
	return result, nil
}

func estimateDraftRequests(doc *article.Document) (int, xRequestEstimate) {
	uniqueMediaCount := countUniqueMedia(doc)
	estimate := xRequestEstimate{
		MediaUpload: uniqueMediaCount,
		CreateDraft: 1,
	}
	estimate.Total = estimate.MediaUpload + estimate.CreateDraft
	return uniqueMediaCount, estimate
}

func countUniqueMedia(doc *article.Document) int {
	if doc == nil {
		return 0
	}
	baseDir := filepath.Dir(doc.SourcePath)
	seen := map[string]struct{}{}
	for _, source := range mediaSources(doc) {
		path := resolveArticlePath(baseDir, source)
		fingerprint, err := mediaFingerprint(path)
		if err != nil {
			fingerprint = "source:" + path
		}
		seen[fingerprint] = struct{}{}
	}
	return len(seen)
}

func mediaSources(doc *article.Document) []string {
	if doc == nil {
		return nil
	}
	var sources []string
	if doc.Cover != "" {
		sources = append(sources, doc.Cover)
	}
	for _, asset := range doc.Assets {
		if asset.Role == "body" {
			sources = append(sources, asset.Source)
		}
	}
	return sources
}

func mediaFingerprint(path string) (string, error) {
	mediaType, size, err := mediafile.ValidateImageFile(path)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash image: %w", err)
	}
	return fmt.Sprintf("%s:%d:%s", mediaType, size, hex.EncodeToString(hash.Sum(nil))), nil
}
