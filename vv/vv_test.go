package vv_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/vv"
)

// TestDeclarationConstants verifies that the VandVFile constant and Declaration
// type exist and are exported correctly.
//
//fusa:test REQ-FO-VV001
func TestDeclarationConstants(t *testing.T) {
	if vv.VandVFile == "" {
		t.Error("VandVFile must not be empty")
	}
	d := vv.Declaration{
		Project:                 "test-project",
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	if d.Project != "test-project" {
		t.Errorf("Project: got %q, want test-project", d.Project)
	}
	if d.ImplementationAuthor != "Alice" {
		t.Errorf("ImplementationAuthor: got %q, want Alice", d.ImplementationAuthor)
	}
	if d.IndependentReviewer != "Bob" {
		t.Errorf("IndependentReviewer: got %q, want Bob", d.IndependentReviewer)
	}
	if d.IndependentTestExecutor != "Carol" {
		t.Errorf("IndependentTestExecutor: got %q, want Carol", d.IndependentTestExecutor)
	}
}

// TestAchievableASIL_NoIndependence verifies that a declaration with no reviewer
// or test executor returns IndependenceLevel 0 and achievable ASIL-B.
//
//fusa:test REQ-FO-VV002
func TestAchievableASIL_NoIndependence(t *testing.T) {
	d := vv.Declaration{
		Project:              "acme",
		ImplementationAuthor: "Alice",
	}
	if got := vv.IndependenceLevel(d); got != 0 {
		t.Errorf("IndependenceLevel: got %d, want 0", got)
	}
	if got := vv.AchievableASIL(d); got != "ASIL-B" {
		t.Errorf("AchievableASIL: got %q, want ASIL-B", got)
	}
}

// TestAchievableASIL_ReviewerOnly verifies that a declaration with an independent
// reviewer (but no test executor) returns IndependenceLevel 1 and achievable ASIL-C.
//
//fusa:test REQ-FO-VV002
func TestAchievableASIL_ReviewerOnly(t *testing.T) {
	d := vv.Declaration{
		Project:              "acme",
		ImplementationAuthor: "Alice",
		IndependentReviewer:  "Bob",
	}
	if got := vv.IndependenceLevel(d); got != 1 {
		t.Errorf("IndependenceLevel: got %d, want 1", got)
	}
	if got := vv.AchievableASIL(d); got != "ASIL-C" {
		t.Errorf("AchievableASIL: got %q, want ASIL-C", got)
	}
}

// TestAchievableASIL_FullIndependence verifies that a declaration with both an
// independent reviewer and test executor returns IndependenceLevel 2 and
// achievable ASIL-D.
//
//fusa:test REQ-FO-VV002
func TestAchievableASIL_FullIndependence(t *testing.T) {
	d := vv.Declaration{
		Project:                 "acme",
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	if got := vv.IndependenceLevel(d); got != 2 {
		t.Errorf("IndependenceLevel: got %d, want 2", got)
	}
	if got := vv.AchievableASIL(d); got != "ASIL-D" {
		t.Errorf("AchievableASIL: got %q, want ASIL-D", got)
	}
}

// TestAchievableASIL_NoAuthor verifies that an empty author still yields the
// correct independence level from the reviewer/executor fields.
//
//fusa:test REQ-FO-VV002
func TestAchievableASIL_NoAuthor(t *testing.T) {
	d := vv.Declaration{
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	if got := vv.IndependenceLevel(d); got != 2 {
		t.Errorf("IndependenceLevel: got %d, want 2", got)
	}
	if got := vv.AchievableASIL(d); got != "ASIL-D" {
		t.Errorf("AchievableASIL: got %q, want ASIL-D", got)
	}
}

// TestValidate_ReviewerSameAsAuthor verifies that Validate warns when the
// reviewer is the same person as the author.
//
//fusa:test REQ-FO-VV003
func TestValidate_ReviewerSameAsAuthor(t *testing.T) {
	d := vv.Declaration{
		ImplementationAuthor: "Alice",
		IndependentReviewer:  "Alice",
	}
	issues := vv.Validate(d)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "independentReviewer") && strings.Contains(iss, "same person") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reviewer-same-as-author warning, got: %v", issues)
	}
}

// TestValidate_ExecutorSameAsAuthor verifies that Validate warns when the
// test executor is the same person as the author.
//
//fusa:test REQ-FO-VV003
func TestValidate_ExecutorSameAsAuthor(t *testing.T) {
	d := vv.Declaration{
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Alice",
	}
	issues := vv.Validate(d)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "independentTestExecutor") && strings.Contains(iss, "same person") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected executor-same-as-author warning, got: %v", issues)
	}
}

// TestValidate_ExecutorWithoutReviewer verifies that Validate warns when a test
// executor is set but no reviewer is set.
//
//fusa:test REQ-FO-VV003
func TestValidate_ExecutorWithoutReviewer(t *testing.T) {
	d := vv.Declaration{
		ImplementationAuthor:    "Alice",
		IndependentTestExecutor: "Carol",
	}
	issues := vv.Validate(d)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "independentTestExecutor") && strings.Contains(iss, "prerequisite") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected prerequisite warning, got: %v", issues)
	}
}

// TestValidate_EmptyAuthor verifies that Validate warns when implementationAuthor
// is empty.
//
//fusa:test REQ-FO-VV003
func TestValidate_EmptyAuthor(t *testing.T) {
	d := vv.Declaration{
		IndependentReviewer: "Bob",
	}
	issues := vv.Validate(d)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "implementationAuthor") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty-author warning, got: %v", issues)
	}
}

// TestValidate_CleanDeclaration verifies that a fully-populated, consistent
// declaration produces no validation issues.
//
//fusa:test REQ-FO-VV003
func TestValidate_CleanDeclaration(t *testing.T) {
	d := vv.Declaration{
		Project:                 "acme",
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	issues := vv.Validate(d)
	if len(issues) != 0 {
		t.Errorf("expected no issues for clean declaration, got: %v", issues)
	}
}

// TestRenderText verifies that Render("text") produces human-readable output
// including the achievable ASIL.
//
//fusa:test REQ-FO-VV004
func TestRenderText(t *testing.T) {
	d := vv.Declaration{
		Project:                 "acme",
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	var buf bytes.Buffer
	if err := vv.Render(&buf, d, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"ASIL-D", "Alice", "Bob", "Carol", "acme"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

// TestRenderText_EmptyFormat verifies that an empty format string defaults to text.
//
//fusa:test REQ-FO-VV004
func TestRenderText_EmptyFormat(t *testing.T) {
	d := vv.Declaration{ImplementationAuthor: "Alice"}
	var buf bytes.Buffer
	if err := vv.Render(&buf, d, ""); err != nil {
		t.Fatalf("Render empty format: %v", err)
	}
	if !strings.Contains(buf.String(), "ASIL-B") {
		t.Errorf("expected ASIL-B in text output:\n%s", buf.String())
	}
}

// TestRenderJSON verifies that Render("json") produces valid JSON containing
// the derived independenceLevel and achievableAsil fields.
//
//fusa:test REQ-FO-VV004
func TestRenderJSON(t *testing.T) {
	d := vv.Declaration{
		Project:              "acme",
		ImplementationAuthor: "Alice",
		IndependentReviewer:  "Bob",
	}
	var buf bytes.Buffer
	if err := vv.Render(&buf, d, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v\noutput: %s", err, buf.String())
	}
	if got["achievableAsil"] != "ASIL-C" {
		t.Errorf("achievableAsil: got %v, want ASIL-C", got["achievableAsil"])
	}
	if lvl, ok := got["independenceLevel"].(float64); !ok || int(lvl) != 1 {
		t.Errorf("independenceLevel: got %v, want 1", got["independenceLevel"])
	}
	if got["implementationAuthor"] != "Alice" {
		t.Errorf("implementationAuthor: got %v, want Alice", got["implementationAuthor"])
	}
}

// TestRenderUnsupportedFormat verifies that Render returns an error for an
// unknown format string.
//
//fusa:test REQ-FO-VV004
func TestRenderUnsupportedFormat(t *testing.T) {
	d := vv.Declaration{}
	var buf bytes.Buffer
	if err := vv.Render(&buf, d, "xml"); err == nil {
		t.Error("expected error for unsupported format xml")
	}
}
