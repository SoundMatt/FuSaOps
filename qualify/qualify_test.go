package qualify_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/qualify"
	"github.com/SoundMatt/FuSaOps/trace"
)

// fakeAdapter is a minimal adapter.Adapter stub.
type fakeAdapter struct {
	name      string
	lang      fusaops.Language
	available bool
}

func (f *fakeAdapter) Name() string               { return f.name }
func (f *fakeAdapter) Language() fusaops.Language { return f.lang }
func (f *fakeAdapter) Tool() string               { return f.name }
func (f *fakeAdapter) Available() bool            { return f.available }
func (f *fakeAdapter) Detect(root string) (bool, error) {
	return true, nil
}
func (f *fakeAdapter) Check(_ context.Context, _ string) ([]fusaops.Finding, error) {
	return nil, nil
}

// qualifyFake additionally implements adapter.Qualifier.
type qualifyFake struct {
	fakeAdapter
	result *trace.Qualification
	err    error
}

func (q *qualifyFake) Qualify(_ context.Context, _ string) (*trace.Qualification, error) {
	return q.result, q.err
}

//fusa:test REQ-FO-QUAL001
func TestTypes(t *testing.T) {
	cr := qualify.ComponentResult{
		Language:  "go",
		Tool:      "gofusa",
		Available: true,
		Total:     10,
		Passed:    10,
	}
	if cr.AllPassed() != true {
		t.Errorf("AllPassed = false, want true")
	}
	cr2 := qualify.ComponentResult{Skipped: "reason"}
	if cr2.AllPassed() != false {
		t.Errorf("AllPassed = true for skipped, want false")
	}
	cr3 := qualify.ComponentResult{Total: 5, Passed: 4, Failed: 1}
	if cr3.AllPassed() != false {
		t.Errorf("AllPassed = true for failed, want false")
	}

	r := &qualify.Report{Total: 10, Passed: 10}
	if r.HasFailures() {
		t.Error("HasFailures = true, want false")
	}
	r2 := &qualify.Report{Total: 5, Failed: 1}
	if !r2.HasFailures() {
		t.Error("HasFailures = false, want true")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunWithQualifier(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 8, Passed: 8, Failed: 0},
		},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Total != 8 || report.Passed != 8 || report.Failed != 0 {
		t.Errorf("aggregate = {%d %d %d}, want {8 8 0}", report.Total, report.Passed, report.Failed)
	}
	if len(report.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(report.Components))
	}
	if report.Components[0].Skipped != "" {
		t.Errorf("component should not be skipped: %s", report.Components[0].Skipped)
	}
	if report.Hash == "" {
		t.Error("Hash should be set")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunWithFailures(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 5, Passed: 4, Failed: 1},
		},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.HasFailures() {
		t.Error("expected HasFailures = true")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunUnavailable(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: false},
			result:      &trace.Qualification{Total: 5, Passed: 5},
		},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Components[0].Skipped == "" {
		t.Error("expected unavailable adapter to be skipped")
	}
	if report.Total != 0 {
		t.Errorf("aggregate total = %d, want 0 (skipped)", report.Total)
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunNoQualifier(t *testing.T) {
	// Adapter that does NOT implement adapter.Qualifier.
	adapters := []adapter.Adapter{
		&fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Components[0].Skipped == "" {
		t.Error("expected non-qualifier adapter to be skipped")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunQualifyError(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			err:         errors.New("tool not found"),
		},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Components[0].Skipped == "" {
		t.Error("expected qualify error to result in skipped component")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunMultipleAdapters(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 10, Passed: 10, Failed: 0},
		},
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "cfusa", lang: fusaops.LangC, available: true},
			result:      &trace.Qualification{Total: 6, Passed: 5, Failed: 1},
		},
	}
	report, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Total != 16 || report.Passed != 15 || report.Failed != 1 {
		t.Errorf("aggregate = {%d %d %d}, want {16 15 1}", report.Total, report.Passed, report.Failed)
	}
}

//fusa:test REQ-FO-QUAL003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	r := &qualify.Report{
		ProjectRoot:            "/proj",
		QualificationType:      "independent",
		QualificationRecordUri: "https://cert.example",
		Total:                  5,
		Passed:                 5,
		Components: []qualify.ComponentResult{
			{Language: "go", Tool: "gofusa", Available: true, Total: 5, Passed: 5},
		},
	}
	path := filepath.Join(dir, qualify.ReportFile)
	if err := qualify.Save(path, r); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var check qualify.Report
	if unmarshalErr := json.Unmarshal(data, &check); unmarshalErr != nil {
		t.Fatalf("Unmarshal: %v", unmarshalErr)
	}

	loaded, err := qualify.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ProjectRoot != "/proj" {
		t.Errorf("ProjectRoot = %q, want /proj", loaded.ProjectRoot)
	}
	if loaded.Total != 5 || loaded.Passed != 5 {
		t.Errorf("Total/Passed = %d/%d, want 5/5", loaded.Total, loaded.Passed)
	}
	if loaded.QualificationType != "independent" {
		t.Errorf("QualificationType = %q, want \"independent\"", loaded.QualificationType)
	}
	if loaded.QualificationRecordUri != "https://cert.example" {
		t.Errorf("QualificationRecordUri = %q, want \"https://cert.example\"", loaded.QualificationRecordUri)
	}
}

//fusa:test REQ-FO-QUAL003
func TestLoadMissing(t *testing.T) {
	_, err := qualify.Load(filepath.Join(t.TempDir(), qualify.ReportFile))
	if !errors.Is(err, fusaops.ErrNoConfig) {
		t.Errorf("expected ErrNoConfig, got %v", err)
	}
}

//fusa:test REQ-FO-QUAL003
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, qualify.ReportFile)
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := qualify.Load(path)
	if err == nil {
		t.Fatal("expected error on bad JSON, got nil")
	}
}

//fusa:test REQ-FO-QUAL004
func TestRenderText(t *testing.T) {
	r := &qualify.Report{
		ProjectRoot: "/proj",
		Total:       10,
		Passed:      9,
		Failed:      1,
		Components: []qualify.ComponentResult{
			{Language: "go", Tool: "gofusa", Available: true, Total: 6, Passed: 6, Failed: 0},
			{Language: "c", Tool: "cfusa", Available: true, Total: 4, Passed: 3, Failed: 1},
			{Language: "rust", Tool: "rsfusa", Available: false, Skipped: "tool not installed"},
		},
	}
	var buf bytes.Buffer
	if err := qualify.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "10 total") {
		t.Errorf("expected '10 total' in output:\n%s", out)
	}
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output:\n%s", out)
	}
	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL in output:\n%s", out)
	}
	if !strings.Contains(out, "skipped") {
		t.Errorf("expected 'skipped' in output:\n%s", out)
	}
}

//fusa:test REQ-FO-QUAL004
func TestRenderJSON(t *testing.T) {
	r := &qualify.Report{
		ProjectRoot: "/proj",
		Total:       5,
		Passed:      5,
		Components:  []qualify.ComponentResult{{Language: "go", Tool: "gofusa", Available: true, Total: 5, Passed: 5}},
	}
	var buf bytes.Buffer
	if err := qualify.Render(&buf, r, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check qualify.Report
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal json output: %v", err)
	}
	if check.ProjectRoot != "/proj" {
		t.Errorf("ProjectRoot = %q, want /proj", check.ProjectRoot)
	}
}

//fusa:test REQ-FO-QUAL004
func TestRenderDefault(t *testing.T) {
	r := &qualify.Report{Components: []qualify.ComponentResult{}}
	var buf bytes.Buffer
	if err := qualify.Render(&buf, r, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if !strings.Contains(buf.String(), "Overall:") {
		t.Errorf("expected 'Overall:' in text output:\n%s", buf.String())
	}
}

//fusa:test REQ-FO-QUAL004
func TestRenderUnknownFormat(t *testing.T) {
	r := &qualify.Report{}
	var buf bytes.Buffer
	if err := qualify.Render(&buf, r, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

//fusa:test REQ-FO-QUAL002
func TestRunHashSet(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 3, Passed: 3},
		},
	}
	r, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(r.Hash, "sha256:") {
		t.Errorf("Hash = %q, want sha256: prefix", r.Hash)
	}
	hashHex := strings.TrimPrefix(r.Hash, "sha256:")
	if _, decErr := hex.DecodeString(hashHex); decErr != nil {
		t.Errorf("Hash hex invalid: %v", decErr)
	}
}

//fusa:test REQ-FO-QUAL005
func TestQualificationTypeConstants(t *testing.T) {
	if qualify.QualificationTypeSelf != "self" {
		t.Errorf("QualificationTypeSelf = %q, want \"self\"", qualify.QualificationTypeSelf)
	}
	if qualify.QualificationTypeIndependent != "independent" {
		t.Errorf("QualificationTypeIndependent = %q, want \"independent\"", qualify.QualificationTypeIndependent)
	}
}

//fusa:test REQ-FO-QUAL006
func TestRunWithOptions(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 4, Passed: 4},
		},
	}
	r, err := qualify.Run(context.Background(), adapters, "/proj",
		qualify.RunOptions{
			Type:      qualify.QualificationTypeIndependent,
			RecordUri: "https://example.com/cert.pdf",
		})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.QualificationType != "independent" {
		t.Errorf("QualificationType = %q, want \"independent\"", r.QualificationType)
	}
	if r.QualificationRecordUri != "https://example.com/cert.pdf" {
		t.Errorf("QualificationRecordUri = %q, want \"https://example.com/cert.pdf\"", r.QualificationRecordUri)
	}
	if !strings.HasPrefix(r.Hash, "sha256:") {
		t.Errorf("Hash = %q, want sha256: prefix", r.Hash)
	}
}

//fusa:test REQ-FO-QUAL006
func TestRunDefaultsToSelf(t *testing.T) {
	adapters := []adapter.Adapter{
		&qualifyFake{
			fakeAdapter: fakeAdapter{name: "gofusa", lang: fusaops.LangGo, available: true},
			result:      &trace.Qualification{Total: 2, Passed: 2},
		},
	}
	r, err := qualify.Run(context.Background(), adapters, "/proj")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.QualificationType != "self" {
		t.Errorf("QualificationType = %q, want \"self\"", r.QualificationType)
	}
}

//fusa:test REQ-FO-QUAL007
func TestRenderTextShowsTypeAndRecord(t *testing.T) {
	r := &qualify.Report{
		ProjectRoot:            "/proj",
		QualificationType:      "independent",
		QualificationRecordUri: "https://example.com/cert.pdf",
		Total:                  5,
		Passed:                 5,
		Components:             []qualify.ComponentResult{},
	}
	var buf bytes.Buffer
	if err := qualify.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Type:") {
		t.Errorf("expected 'Type:' in text output:\n%s", out)
	}
	if !strings.Contains(out, "Record:") {
		t.Errorf("expected 'Record:' in text output:\n%s", out)
	}
	if !strings.Contains(out, "independent") {
		t.Errorf("expected 'independent' in text output:\n%s", out)
	}
	if !strings.Contains(out, "https://example.com/cert.pdf") {
		t.Errorf("expected record URI in text output:\n%s", out)
	}
}
