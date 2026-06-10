package fusaops

import (
	"errors"
	"strings"
	"testing"
)

//fusa:test REQ-FO-CORE002
func TestSeverityRank(t *testing.T) {
	if SeverityError.Rank() <= SeverityWarning.Rank() {
		t.Errorf("error rank %d not greater than warning rank %d", SeverityError.Rank(), SeverityWarning.Rank())
	}
	if SeverityWarning.Rank() <= SeverityInfo.Rank() {
		t.Errorf("warning rank %d not greater than info rank %d", SeverityWarning.Rank(), SeverityInfo.Rank())
	}
	if SeverityInfo.Rank() <= Severity("bogus").Rank() {
		t.Errorf("info rank should exceed unknown severity rank")
	}
}

//fusa:test REQ-FO-CORE001
func TestSeverityString(t *testing.T) {
	if SeverityError.String() != "ERROR" {
		t.Errorf("got %q, want ERROR", SeverityError.String())
	}
}

//fusa:test REQ-FO-CORE003
func TestLanguageString(t *testing.T) {
	cases := map[Language]string{LangGo: "go", LangC: "c", LangCpp: "cpp"}
	for l, want := range cases {
		if l.String() != want {
			t.Errorf("language %v: got %q, want %q", l, l.String(), want)
		}
	}
}

//fusa:test REQ-FO-CORE004
//fusa:test REQ-FO-CORE005
func TestFindingAndLocation(t *testing.T) {
	f := Finding{
		Language: LangGo, Tool: "gofusa", RuleID: "X1", Severity: SeverityError,
		Message: "m", Location: Location{File: "a.go", Line: 7, Column: 3}, Remediation: "fix",
	}
	if f.Language != LangGo || f.Tool != "gofusa" {
		t.Errorf("finding not attributable to language/tool: %+v", f)
	}
	if f.Location.File != "a.go" || f.Location.Line != 7 || f.Location.Column != 3 {
		t.Errorf("location wrong: %+v", f.Location)
	}
}

//fusa:test REQ-FO-ERR001
//fusa:test REQ-FO-ERR002
//fusa:test REQ-FO-ERR003
//fusa:test REQ-FO-ERR004
//fusa:test REQ-FO-ERR005
func TestSentinelErrors(t *testing.T) {
	// Each sentinel must be distinct and comparable with errors.Is when wrapped.
	for _, e := range []error{ErrNoConfig, ErrInvalidConfig, ErrNoAdapters, ErrCheckFailed} {
		wrapped := errors.New("ctx: " + e.Error())
		if errors.Is(wrapped, e) {
			t.Errorf("unrelated error should not match %v", e)
		}
		if !errors.Is(e, e) {
			t.Errorf("%v should match itself", e)
		}
	}
	if errors.Is(ErrNoConfig, ErrInvalidConfig) {
		t.Error("distinct sentinels must not be equal")
	}
}

// TestComputeFingerprint verifies the §4.2 fingerprint algorithm.
//
//fusa:test REQ-FO-CORE006
func TestComputeFingerprint(t *testing.T) {
	f := Finding{
		RuleID:  "LINT001",
		Message: "function has 42 lines",
		Location: Location{File: "main.go"},
	}
	fp := ComputeFingerprint(f)
	if !strings.HasPrefix(fp, "sha256:") {
		t.Errorf("fingerprint must start with sha256:, got %q", fp)
	}
	if len(fp) != len("sha256:")+64 {
		t.Errorf("fingerprint length wrong: %d", len(fp))
	}
	// Deterministic: same input → same output.
	if ComputeFingerprint(f) != fp {
		t.Error("ComputeFingerprint must be deterministic")
	}
	// Digit normalisation: "42 lines" and "7 lines" should produce the same fingerprint.
	f2 := f
	f2.Message = "function has 7 lines"
	if ComputeFingerprint(f2) != fp {
		t.Errorf("digit normalisation failed: %q vs %q", fp, ComputeFingerprint(f2))
	}
	// Different rule → different fingerprint.
	f3 := f
	f3.RuleID = "LINT002"
	if ComputeFingerprint(f3) == fp {
		t.Error("different ruleId must produce different fingerprint")
	}
	// Different file → different fingerprint.
	f4 := f
	f4.Location.File = "other.go"
	if ComputeFingerprint(f4) == fp {
		t.Error("different file must produce different fingerprint")
	}
}
