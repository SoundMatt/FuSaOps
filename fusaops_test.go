package fusaops

import (
	"errors"
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
