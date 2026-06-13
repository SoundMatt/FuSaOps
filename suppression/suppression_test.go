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
