package e2e

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestInspectJSON(t *testing.T) {
	cmd := md2xCommand(t, "inspect", "../testdata/articles/formatting.md", "--json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("md2x inspect --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success bool `json:"success"`
		Data    struct {
			Title      string `json:"title"`
			BlockCount int    `json:"block_count"`
			Ready      bool   `json:"ready"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal inspect json: %v\nstdout:\n%s", err, stdout.String())
	}

	if !got.Success {
		t.Fatalf("success = false, want true")
	}
	if got.Data.Title != "Formatting" {
		t.Fatalf("data.title = %q, want %q", got.Data.Title, "Formatting")
	}
	if got.Data.BlockCount <= 0 {
		t.Fatalf("data.block_count = %d, want > 0", got.Data.BlockCount)
	}
	if !got.Data.Ready {
		t.Fatalf("data.ready = false, want true")
	}
}

func TestRenderDraftJSJSON(t *testing.T) {
	cmd := md2xCommand(t, "render", "../testdata/articles/formatting.md", "--format", "draftjs", "--json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("md2x render --format draftjs --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success bool `json:"success"`
		Data    struct {
			ContentState struct {
				Blocks []struct {
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"blocks"`
			} `json:"content_state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal render json: %v\nstdout:\n%s", err, stdout.String())
	}

	if !got.Success {
		t.Fatalf("success = false, want true")
	}
	if len(got.Data.ContentState.Blocks) == 0 {
		t.Fatalf("data.content_state.blocks is empty")
	}
}

func TestRenderDraftJSDefaultWritesRawContentStateJSON(t *testing.T) {
	cmd := md2xCommand(t, "render", "../testdata/articles/formatting.md", "--format", "draftjs")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("md2x render --format draftjs failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success       *bool  `json:"success"`
		SchemaVersion string `json:"schema_version"`
		Data          any    `json:"data"`
		Blocks        []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"blocks"`
		Entities []any `json:"entities"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal render default json: %v\nstdout:\n%s", err, stdout.String())
	}
	if got.Success != nil || got.SchemaVersion != "" || got.Data != nil {
		t.Fatalf("default render wrote envelope fields: %#v", got)
	}
	if len(got.Blocks) == 0 {
		t.Fatalf("blocks is empty")
	}
	if got.Entities == nil {
		t.Fatalf("entities is nil")
	}
}
