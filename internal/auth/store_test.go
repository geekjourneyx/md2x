package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenStoreSavesWithPrivatePermissionsAndRedactsStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth", "default.json")
	store := FileStore{Path: path}
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	if err := store.Save(Token{
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		TokenType:    "bearer",
		Scope:        "tweet.read tweet.write offline.access",
		ExpiresAt:    expiresAt,
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token file mode = %v, want 0600", got)
	}

	token, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access-secret" || token.RefreshToken != "refresh-secret" || !token.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("loaded token = %#v", token)
	}

	status := RedactTokenStatus(token)
	if status.AccessToken != "<redacted>" || status.RefreshToken != "<redacted>" {
		t.Fatalf("status did not redact secrets: %#v", status)
	}
	if !status.Authenticated || status.ExpiresAt != expiresAt.Format(time.RFC3339) {
		t.Fatalf("status = %#v", status)
	}
}

func TestDefaultStorePathUsesXDGStateHomeAndProfile(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)

	path, err := DefaultStorePath("work")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(stateHome, "md2x", "auth", "work.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestFileStoreDeleteRemovesTokenAndMissingDeleteIsOK(t *testing.T) {
	store := FileStore{Path: filepath.Join(t.TempDir(), "default.json")}
	if err := store.Save(Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("Load err = %v, want ErrNotAuthenticated", err)
	}
	if err := store.Delete(); err != nil {
		t.Fatalf("delete missing token: %v", err)
	}
}
