package qualitybar_test

import (
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/qualitybar"
)

//fusa:test REQ-QB001
func TestDetectPlaceholderBracketText(t *testing.T) {
	fields := []qualitybar.QualField{
		{EntryID: "H-001", Field: "hazard", Value: "[describe hazard]"},
	}
	findings := qualitybar.DetectPlaceholder("test.json", fields)
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.RuleID != "FUSA-STUB001" || f.Severity != fusaops.SeverityError {
		t.Errorf("finding = %+v, want RuleID=FUSA-STUB001 Severity=ERROR", f)
	}
	if f.Fingerprint == "" {
		t.Error("Fingerprint must be set")
	}
}

//fusa:test REQ-QB001
func TestDetectPlaceholderSubstrings(t *testing.T) {
	cases := []string{
		"Example hazard — replace with project-specific hazard",
		"TBD",
		"lorem ipsum dolor",
		"please fill in this section",
	}
	for _, v := range cases {
		fields := []qualitybar.QualField{{EntryID: "X-001", Field: "threat", Value: v}}
		if got := qualitybar.DetectPlaceholder("test.json", fields); len(got) != 1 {
			t.Errorf("DetectPlaceholder(%q): got %d findings, want 1", v, len(got))
		}
	}
}

//fusa:test REQ-QB001
func TestDetectPlaceholderRealContentNotFlagged(t *testing.T) {
	fields := []qualitybar.QualField{
		{EntryID: "FM-001", Field: "failureMode", Value: "Adapter binary not found on PATH"},
	}
	if got := qualitybar.DetectPlaceholder("test.json", fields); len(got) != 0 {
		t.Errorf("real content flagged as placeholder: %+v", got)
	}
}

//fusa:test REQ-QB002
func TestDetectBlankFallbackTriggersOnTemplatedField(t *testing.T) {
	var fields []qualitybar.QualField
	for i := 0; i < 12; i++ {
		fields = append(fields, qualitybar.QualField{
			EntryID: "FM-0", Field: "failureMode", Value: "Returns incorrect value",
		})
	}
	findings := qualitybar.DetectBlankFallback("test.json", fields)
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.RuleID != "FUSA-STUB002" || f.Severity != fusaops.SeverityWarning {
		t.Errorf("finding = %+v, want RuleID=FUSA-STUB002 Severity=WARNING", f)
	}
}

//fusa:test REQ-QB002
func TestDetectBlankFallbackBelowThresholdNotFlagged(t *testing.T) {
	// Only 8 entries — below the >=10 trigger threshold.
	var fields []qualitybar.QualField
	for i := 0; i < 8; i++ {
		fields = append(fields, qualitybar.QualField{
			EntryID: "FM-0", Field: "failureMode", Value: "same text",
		})
	}
	if got := qualitybar.DetectBlankFallback("test.json", fields); len(got) != 0 {
		t.Errorf("expected no findings below the 10-entry threshold, got %+v", got)
	}
}

//fusa:test REQ-QB002
func TestDetectBlankFallbackDiverseContentNotFlagged(t *testing.T) {
	var fields []qualitybar.QualField
	for i := 0; i < 12; i++ {
		fields = append(fields, qualitybar.QualField{
			EntryID: "FM-0", Field: "failureMode",
			Value: strings.Repeat("x", i+1), // 12 distinct values
		})
	}
	if got := qualitybar.DetectBlankFallback("test.json", fields); len(got) != 0 {
		t.Errorf("expected no findings for diverse content, got %+v", got)
	}
}

//fusa:test REQ-QB002
func TestDetectBlankFallbackGroupsBySeparateFields(t *testing.T) {
	var fields []qualitybar.QualField
	for i := 0; i < 12; i++ {
		fields = append(fields, qualitybar.QualField{EntryID: "FM-0", Field: "failureMode", Value: "same"})
		fields = append(fields, qualitybar.QualField{EntryID: "FM-0", Field: "effect", Value: strings.Repeat("y", i+1)})
	}
	findings := qualitybar.DetectBlankFallback("test.json", fields)
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1 (only failureMode should trigger)", len(findings))
	}
	if !strings.Contains(findings[0].Message, "failureMode") {
		t.Errorf("expected finding about failureMode, got %q", findings[0].Message)
	}
}
