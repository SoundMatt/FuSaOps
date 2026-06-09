package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderToFileWritesFile(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	path := filepath.Join(t.TempDir(), "out.json")
	if err := RenderToFile(r, "json", path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"summary\"") {
		t.Errorf("file missing summary: %s", data)
	}
}

func TestRenderToFileStdout(t *testing.T) {
	// Empty path renders to stdout without error.
	r := New("/root", "demo", nil)
	if err := RenderToFile(r, "json", ""); err != nil {
		t.Fatal(err)
	}
}

func TestRenderToFileBadPath(t *testing.T) {
	r := New("/root", "demo", nil)
	if err := RenderToFile(r, "json", filepath.Join(t.TempDir(), "nope", "out.json")); err == nil {
		t.Error("expected error writing to nonexistent directory")
	}
}
