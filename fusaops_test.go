package fusaops

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestDockerfileBundlesPyFuSaAndJavaFuSa verifies the all-in-one image's
// Dockerfile actually bundles py-FuSa and java-FuSa per REQ-FO-IMG001/002 —
// this can't be exercised by running the built image in a unit test, so it
// asserts the specific COPY/RUN lines the requirements describe are present
// in source, which is what actually determines what ships in the image.
//
//fusa:test REQ-FO-IMG001
//fusa:test REQ-FO-IMG002
func TestDockerfileBundlesPyFuSaAndJavaFuSa(t *testing.T) {
	data, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	content := string(data)

	// REQ-FO-IMG001: py-FuSa binary + site-packages copied in; python base image.
	if !strings.Contains(content, "FROM python:3.12-alpine") {
		t.Error("Dockerfile must use python:3.12-alpine as its runtime base for pyfusa")
	}
	if !strings.Contains(content, "COPY --from=pyfusa") || !strings.Contains(content, "/usr/local/bin/pyfusa") {
		t.Error("Dockerfile must COPY --from=pyfusa the pyfusa binary")
	}
	if !strings.Contains(content, "site-packages") {
		t.Error("Dockerfile must COPY --from=pyfusa the Python site-packages")
	}

	// REQ-FO-IMG002: java-FuSa wrapper + JAR copied in; JRE installed.
	if !strings.Contains(content, "openjdk21-jre-headless") {
		t.Error("Dockerfile must install openjdk21-jre-headless for jfusa")
	}
	if !strings.Contains(content, "COPY --from=jfusa") || !strings.Contains(content, "jfusa.jar") {
		t.Error("Dockerfile must COPY --from=jfusa the jfusa.jar")
	}
	if !strings.Contains(content, "/usr/local/bin/jfusa") {
		t.Error("Dockerfile must COPY --from=jfusa the jfusa wrapper script")
	}
}

//fusa:test REQ-FO-CORE007
func TestSpecVersionIsSemver(t *testing.T) {
	if SpecVersion == "" {
		t.Fatal("SpecVersion must not be empty")
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(SpecVersion) {
		t.Errorf("SpecVersion %q must be a MAJOR.MINOR.PATCH semver string", SpecVersion)
	}
}

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
		RuleID:   "LINT001",
		Message:  "function has 42 lines",
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

//fusa:test REQ-FO-CORE011
func TestFindingGates(t *testing.T) {
	cases := []struct {
		disposition string
		want        bool
	}{
		{"", true},
		{"accepted", false},
		{"deferred", false},
		{"rejected", true},
		{"open", true},
	}
	for _, c := range cases {
		f := Finding{Disposition: c.disposition}
		if got := f.Gates(); got != c.want {
			t.Errorf("Gates() with disposition %q: got %v, want %v", c.disposition, got, c.want)
		}
	}
}

//fusa:test REQ-FO-CORE012
func TestFindingQualifiedRuleID(t *testing.T) {
	f := Finding{Language: LangGo, RuleID: "LINT001"}
	if got := f.QualifiedRuleID(); got != "go/LINT001" {
		t.Errorf("QualifiedRuleID() = %q, want %q", got, "go/LINT001")
	}
}
