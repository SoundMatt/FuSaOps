package pr_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SoundMatt/FuSaOps/pr"
)

//fusa:test REQ-FO-PR001
func TestTypes(t *testing.T) {
	r := pr.ProblemReport{
		ID:         "PR-001",
		Title:      "Null pointer in init",
		PhaseFound: pr.PhaseDevelopment,
		Severity:   pr.PRSeverityMajor,
		Status:     pr.StatusOpen,
	}
	if r.ID != "PR-001" || r.Title != "Null pointer in init" || r.PhaseFound != pr.PhaseDevelopment || r.Severity != pr.PRSeverityMajor || r.Status != pr.StatusOpen {
		t.Errorf("ProblemReport fields not set correctly: %+v", r)
	}
	log := &pr.Log{Project: "firmware"}
	if log.Project != "firmware" {
		t.Errorf("Project = %q, want firmware", log.Project)
	}
}

//fusa:test REQ-FO-PR002
func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	log, err := pr.Load(dir)
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if len(log.Reports) != 0 {
		t.Errorf("expected empty log, got %d reports", len(log.Reports))
	}
}

//fusa:test REQ-FO-PR002
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	log := &pr.Log{Project: "myproj", Reports: []pr.ProblemReport{
		{
			ID:         "PR-001",
			Title:      "Test issue",
			PhaseFound: pr.PhaseVerification,
			Severity:   pr.PRSeverityMinor,
			Status:     pr.StatusOpen,
			Created:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Updated:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}}

	if err := pr.Save(dir, log); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists and is valid JSON.
	path := filepath.Join(dir, pr.ProblemsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var check pr.Log
	if unmarshalErr := json.Unmarshal(data, &check); unmarshalErr != nil {
		t.Fatalf("Unmarshal: %v", unmarshalErr)
	}

	loaded, err := pr.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Reports) != 1 {
		t.Errorf("loaded %d reports, want 1", len(loaded.Reports))
	}
	if loaded.Reports[0].ID != "PR-001" {
		t.Errorf("ID = %q, want PR-001", loaded.Reports[0].ID)
	}
}

//fusa:test REQ-FO-PR002
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, pr.ProblemsFile)
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := pr.Load(dir)
	if err == nil {
		t.Fatal("expected error on bad JSON, got nil")
	}
}

//fusa:test REQ-FO-PR003
func TestAdd(t *testing.T) {
	log := &pr.Log{Project: "proj"}
	log = pr.Add(log, pr.ProblemReport{
		ID:       "PR-001",
		Title:    "First issue",
		Severity: pr.PRSeverityMajor,
	})
	if len(log.Reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(log.Reports))
	}
	r := log.Reports[0]
	if r.Status != pr.StatusOpen {
		t.Errorf("default status = %q, want open", r.Status)
	}
	if r.Severity != pr.PRSeverityMajor {
		t.Errorf("severity = %q, want major", r.Severity)
	}
	if r.Updated.IsZero() {
		t.Error("Updated should be set by Add")
	}
}

//fusa:test REQ-FO-PR003
func TestAddDefaultSeverity(t *testing.T) {
	log := &pr.Log{}
	log = pr.Add(log, pr.ProblemReport{ID: "PR-002", Title: "Minor thing"})
	if log.Reports[0].Severity != pr.PRSeverityMinor {
		t.Errorf("default severity = %q, want minor", log.Reports[0].Severity)
	}
}

//fusa:test REQ-FO-PR003
func TestClose(t *testing.T) {
	log := &pr.Log{Reports: []pr.ProblemReport{
		{ID: "PR-001", Status: pr.StatusOpen},
	}}
	if err := pr.Close(log, "PR-001", "fixed in commit abc123"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if log.Reports[0].Status != pr.StatusClosed {
		t.Errorf("status = %q, want closed", log.Reports[0].Status)
	}
	if log.Reports[0].Resolution == "" {
		t.Error("Resolution should be set")
	}
}

//fusa:test REQ-FO-PR003
func TestCloseNotFound(t *testing.T) {
	log := &pr.Log{}
	if err := pr.Close(log, "PR-999", ""); err == nil {
		t.Fatal("expected error for missing ID, got nil")
	}
}

//fusa:test REQ-FO-PR003
func TestFind(t *testing.T) {
	log := &pr.Log{Reports: []pr.ProblemReport{
		{ID: "PR-001", Title: "A"},
		{ID: "PR-002", Title: "B"},
	}}
	r := pr.Find(log, "PR-002")
	if r == nil {
		t.Fatal("Find returned nil for PR-002")
	}
	if r.Title != "B" {
		t.Errorf("title = %q, want B", r.Title)
	}
	if pr.Find(log, "PR-999") != nil {
		t.Error("Find should return nil for unknown ID")
	}
}

//fusa:test REQ-FO-PR004
func TestRenderText(t *testing.T) {
	log := &pr.Log{
		Project: "fw",
		Reports: []pr.ProblemReport{
			{
				ID:         "PR-001",
				Title:      "Stack overflow in ISR",
				PhaseFound: pr.PhaseIntegration,
				Severity:   pr.PRSeverityCritical,
				Status:     pr.StatusOpen,
				Created:    time.Now().UTC(),
				Updated:    time.Now().UTC(),
			},
		},
	}
	var buf bytes.Buffer
	if err := pr.Render(&buf, log, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "PR-001") {
		t.Errorf("text output missing PR-001:\n%s", out)
	}
	if !strings.Contains(out, "Stack overflow") {
		t.Errorf("text output missing title:\n%s", out)
	}
	if !strings.Contains(out, "critical") {
		t.Errorf("text output missing severity:\n%s", out)
	}
}

//fusa:test REQ-FO-PR004
func TestRenderTextWithResolution(t *testing.T) {
	log := &pr.Log{Reports: []pr.ProblemReport{
		{
			ID:         "PR-002",
			Title:      "Memory leak",
			PhaseFound: pr.PhaseDevelopment,
			Severity:   pr.PRSeverityMajor,
			Status:     pr.StatusClosed,
			Resolution: "freed buffer in cleanup path",
			Created:    time.Now().UTC(),
			Updated:    time.Now().UTC(),
		},
	}}
	var buf bytes.Buffer
	if err := pr.Render(&buf, log, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	if !strings.Contains(buf.String(), "freed buffer") {
		t.Errorf("resolution not shown in output:\n%s", buf.String())
	}
}

//fusa:test REQ-FO-PR004
func TestRenderJSON(t *testing.T) {
	log := &pr.Log{Project: "p", Reports: []pr.ProblemReport{
		{ID: "PR-001", Title: "T", Severity: pr.PRSeverityMinor, Status: pr.StatusOpen},
	}}
	var buf bytes.Buffer
	if err := pr.Render(&buf, log, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check pr.Log
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal json output: %v", err)
	}
	if check.Project != "p" {
		t.Errorf("project = %q, want p", check.Project)
	}
}

//fusa:test REQ-FO-PR004
func TestRenderEmpty(t *testing.T) {
	log := &pr.Log{}
	var buf bytes.Buffer
	if err := pr.Render(&buf, log, "text"); err != nil {
		t.Fatalf("Render empty: %v", err)
	}
	if !strings.Contains(buf.String(), "0 total") {
		t.Errorf("expected 0 total: %s", buf.String())
	}
}

//fusa:test REQ-FO-PR004
func TestRenderUnknownFormat(t *testing.T) {
	log := &pr.Log{}
	var buf bytes.Buffer
	if err := pr.Render(&buf, log, "xml"); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// TestSaveWriteError verifies Save returns an error when the project root
// directory does not exist.
//
//fusa:test REQ-FO-PR002
func TestSaveWriteError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if err := pr.Save(root, &pr.Log{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
