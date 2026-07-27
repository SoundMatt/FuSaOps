package metrics

// Gap test for metrics.go: Load JSON parse error (line 60-62). The read error
// path (line 57) is already covered by TestLoadReadError in metrics_test.go.
// The Save MarshalIndent error (line 72-74) is unreachable for TimeSeries.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadBadJSONGap verifies Load returns a non-nil error when the metrics
// file contains invalid JSON, covering metrics.go:60.50,62.3.
//
//fusa:test REQ-FO-MET002
func TestLoadBadJSONGap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, MetricsFile), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load: expected error for malformed JSON, got nil")
	}
}
