package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const testGoCache = "/tmp/md2x-go-build"

func md2xBinary(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "md2x")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "../cmd/md2x")
	cmd.Env = append(os.Environ(), "GOCACHE="+testGoCache)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build md2x test binary: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return bin
}

func md2xCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(md2xBinary(t), args...)
	cmd.Env = append(
		os.Environ(),
		"GOCACHE="+testGoCache,
		"X_BEARER_TOKEN=",
		"XDG_CONFIG_HOME="+filepath.Join(t.TempDir(), "config"),
		"XDG_STATE_HOME="+filepath.Join(t.TempDir(), "state"),
	)
	return cmd
}

func TestVersionJSON(t *testing.T) {
	cmd := md2xCommand(t, "version", "--json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("md2x version --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success       bool   `json:"success"`
		SchemaVersion string `json:"schema_version"`
		Code          string `json:"code"`
		Data          struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal version json: %v\nstdout:\n%s", err, stdout.String())
	}

	if !got.Success {
		t.Fatalf("success = false, want true")
	}
	if got.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want %q", got.SchemaVersion, "v1")
	}
	if got.Code != "OK" {
		t.Fatalf("code = %q, want %q", got.Code, "OK")
	}
	if got.Data.Version == "" {
		t.Fatalf("data.version is empty")
	}
}

func TestUnknownCommandJSONErrorWritesEnvelopeToStdout(t *testing.T) {
	cmd := md2xCommand(t, "--json", "unknown")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("md2x --json unknown succeeded, want non-zero exit")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success       bool        `json:"success"`
		SchemaVersion string      `json:"schema_version"`
		Status        string      `json:"status"`
		Code          string      `json:"code"`
		Message       string      `json:"message"`
		Error         interface{} `json:"error"`
	}
	if err := json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&got); err != nil {
		t.Fatalf("decode stdout json: %v\nstdout:\n%s", err, stdout.String())
	}
	if got.Success {
		t.Fatalf("success = true, want false")
	}
	if got.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want %q", got.SchemaVersion, "v1")
	}
	if got.Status != "failed" {
		t.Fatalf("status = %q, want %q", got.Status, "failed")
	}
	if got.Code == "" {
		t.Fatalf("code is empty")
	}
	if got.Message == "" {
		t.Fatalf("message is empty")
	}
	if got.Error == nil {
		t.Fatalf("error is nil")
	}
}
