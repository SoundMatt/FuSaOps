package suppression_test

// Gap tests covering uncovered branches in suppression.go.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/suppression"
)

// TestLoadConfigBadJSON verifies LoadConfig returns an error when the suppress
// file exists but contains malformed JSON, covering
// suppression.go:47.51,49.3 (json.Unmarshal error path).
//
//fusa:test REQ-FO-SUP002
func TestLoadConfigBadJSON(t *testing.T) {
	dir := t.TempDir()
	suppFile := filepath.Join(dir, ".fusaops-suppress.json")
	if err := os.WriteFile(suppFile, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := suppression.LoadConfig(suppFile)
	if err == nil {
		t.Error("LoadConfig: expected error for malformed JSON, got nil")
	}
}
