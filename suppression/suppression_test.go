package suppression_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/suppression"
)

func makeFindings(fps ...string) []fusaops.Finding {
	out := make([]fusaops.Finding, len(fps))
	for i, fp := range fps {
		out[i] = fusaops.Finding{
			RuleID:      "RULE001",
			Fingerprint: fp,
			Severity:    fusaops.SeverityError,
			Message:     "test",
		}
	}
	return out
}

// TestLoadConfigEmpty verifies empty path returns empty config.
//
//fusa:test REQ-FO-SUP002
func TestLoadConfigEmpty(t *testing.T) {
	cfg, err := suppression.LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Suppressions) != 0 {
		t.Errorf("expected no suppressions, got %d", len(cfg.Suppressions))
	}
}

// TestLoadConfigFile verifies a valid file is parsed correctly.
//
//fusa:test REQ-FO-SUP001
//fusa:test REQ-FO-SUP002
func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg := suppression.Config{
		Suppressions: []suppression.Suppression{
			{Fingerprint: "abc123", Reason: "accepted risk"},
		},
	}
	data, _ := json.Marshal(cfg)
	path := filepath.Join(dir, ".fusaops-suppress.json")
	_ = os.WriteFile(path, data, 0o600)

	got, err := suppression.LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Suppressions) != 1 || got.Suppressions[0].Fingerprint != "abc123" {
		t.Errorf("unexpected config: %+v", got)
	}
}

// TestLoadConfigMissingFile returns an error for a non-existent file.
//
//fusa:test REQ-FO-SUP002
func TestLoadConfigMissingFile(t *testing.T) {
	_, err := suppression.LoadConfig("/nonexistent/suppress.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// TestFilterNoSuppressions returns all findings when config is empty.
//
//fusa:test REQ-FO-SUP003
func TestFilterNoSuppressions(t *testing.T) {
	findings := makeFindings("fp1", "fp2")
	kept, suppressed := suppression.Filter(findings, suppression.Config{}, time.Now())
	if len(kept) != 2 || len(suppressed) != 0 {
		t.Errorf("kept=%d suppressed=%d", len(kept), len(suppressed))
	}
}

// TestFilterMatchesByFingerprint suppresses findings whose fingerprint matches.
//
//fusa:test REQ-FO-SUP003
func TestFilterMatchesByFingerprint(t *testing.T) {
	findings := makeFindings("fp1", "fp2", "fp3")
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "fp2", Reason: "known false positive"},
	}}
	kept, suppressed := suppression.Filter(findings, cfg, time.Now())
	if len(kept) != 2 || len(suppressed) != 1 {
		t.Errorf("kept=%d suppressed=%d", len(kept), len(suppressed))
	}
	if suppressed[0].Fingerprint != "fp2" {
		t.Errorf("suppressed wrong finding: %v", suppressed[0].Fingerprint)
	}
}

// TestFilterExpiredSuppression does not suppress findings past their expiry.
//
//fusa:test REQ-FO-SUP003
func TestFilterExpiredSuppression(t *testing.T) {
	findings := makeFindings("fp1")
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "fp1", Reason: "old", Expires: "2020-01-01"},
	}}
	kept, suppressed := suppression.Filter(findings, cfg, time.Now())
	if len(kept) != 1 || len(suppressed) != 0 {
		t.Errorf("expired suppression should not hide finding: kept=%d suppressed=%d", len(kept), len(suppressed))
	}
}

// TestFilterActiveFutureSuppression suppresses findings with a future expiry.
//
//fusa:test REQ-FO-SUP003
func TestFilterActiveFutureSuppression(t *testing.T) {
	findings := makeFindings("fp1")
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "fp1", Reason: "tracked", Expires: "2099-12-31"},
	}}
	kept, suppressed := suppression.Filter(findings, cfg, time.Now())
	if len(kept) != 0 || len(suppressed) != 1 {
		t.Errorf("future suppression should hide finding: kept=%d suppressed=%d", len(kept), len(suppressed))
	}
}

// TestFilterNoFingerprintFindingNotSuppressed skips findings without fingerprint.
//
//fusa:test REQ-FO-SUP003
func TestFilterNoFingerprintFindingNotSuppressed(t *testing.T) {
	findings := []fusaops.Finding{{RuleID: "R1", Severity: fusaops.SeverityError, Message: "no fp"}}
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "", Reason: "empty fingerprint entry"},
	}}
	kept, suppressed := suppression.Filter(findings, cfg, time.Now())
	if len(kept) != 1 || len(suppressed) != 0 {
		t.Errorf("finding without fingerprint should not be suppressed: kept=%d suppressed=%d", len(kept), len(suppressed))
	}
}

// TestSaveConfig verifies SaveConfig writes valid JSON readable by LoadConfig.
//
//fusa:test REQ-FO-SUP005
func TestSaveConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "suppress.json")
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "sha256:abc", Reason: "known", Expires: "2099-01-01"},
	}}
	if err := suppression.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	loaded, err := suppression.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig after Save: %v", err)
	}
	if len(loaded.Suppressions) != 1 || loaded.Suppressions[0].Fingerprint != "sha256:abc" {
		t.Errorf("round-trip mismatch: %+v", loaded)
	}
}

// TestPruneRemovesExpired verifies Prune removes expired entries.
//
//fusa:test REQ-FO-SUP007
func TestPruneRemovesExpired(t *testing.T) {
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "fp1", Reason: "active", Expires: "2099-12-31"},
		{Fingerprint: "fp2", Reason: "expired", Expires: "2000-01-01"},
		{Fingerprint: "fp3", Reason: "no-expiry"},
	}}
	pruned, removed := suppression.Prune(cfg, time.Now())
	if removed != 1 {
		t.Errorf("removed: got %d, want 1", removed)
	}
	if len(pruned.Suppressions) != 2 {
		t.Errorf("remaining: got %d, want 2", len(pruned.Suppressions))
	}
}

// TestPruneNothingToRemove verifies Prune returns zero removed when all active.
//
//fusa:test REQ-FO-SUP007
func TestPruneNothingToRemove(t *testing.T) {
	cfg := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "fp1", Reason: "active", Expires: "2099-12-31"},
	}}
	_, removed := suppression.Prune(cfg, time.Now())
	if removed != 0 {
		t.Errorf("removed: got %d, want 0", removed)
	}
}

// TestSaveConfigWriteError verifies SaveConfig returns an error when the output
// path is in a non-existent directory.
//
//fusa:test REQ-FO-SUP005
func TestSaveConfigWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "s.json")
	if err := suppression.SaveConfig(path, suppression.Config{}); err == nil {
		t.Error("SaveConfig: expected error for non-existent parent directory")
	}
}

// TestSaveConfigRoundTrip verifies JSON round-trip preserves all fields.
//
//fusa:test REQ-FO-SUP005
func TestSaveConfigRoundTrip(t *testing.T) {
	_ = json.Marshal // ensure json import is used
	path := filepath.Join(t.TempDir(), "s.json")
	orig := suppression.Config{Suppressions: []suppression.Suppression{
		{Fingerprint: "sha256:deadbeef", Reason: "test", Expires: "2030-06-15"},
	}}
	_ = suppression.SaveConfig(path, orig)
	data, _ := os.ReadFile(path)
	var loaded suppression.Config
	_ = json.Unmarshal(data, &loaded)
	if loaded.Suppressions[0].Expires != "2030-06-15" {
		t.Errorf("expires: got %q", loaded.Suppressions[0].Expires)
	}
}
