// Package pr implements the DO-178C §11.17 Software Problem Report workflow
// for FuSaOps multi-language projects.
//
// Problem reports are stored in .fusaops-problems.json alongside the codebase
// as a lifecycle data item. Each entry records an ID, title, phase found,
// severity, status, and optional resolution.
//
// Usage:
//
//	log, err := pr.Load(projectRoot)
//	log = pr.Add(log, pr.ProblemReport{...})
//	err = pr.Save(projectRoot, log)
package pr

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ProblemsFile is the default filename for the problem report log.
const ProblemsFile = ".fusaops-problems.json"

// Phase identifies the software lifecycle phase in which the problem was found
// or fixed.
//
//fusa:req REQ-FO-PR001
type Phase string

const (
	PhasePlanning     Phase = "planning"
	PhaseDevelopment  Phase = "development"
	PhaseVerification Phase = "verification"
	PhaseIntegration  Phase = "integration"
	PhaseOperation    Phase = "operation"
)

// Status is the current state of a problem report.
//
//fusa:req REQ-FO-PR001
type Status string

const (
	StatusOpen     Status = "open"
	StatusInWork   Status = "in-work"
	StatusClosed   Status = "closed"
	StatusDeferred Status = "deferred"
)

// PRSeverity is the impact classification of the problem.
//
//fusa:req REQ-FO-PR001
type PRSeverity string

const (
	PRSeverityCritical PRSeverity = "critical"
	PRSeverityMajor    PRSeverity = "major"
	PRSeverityMinor    PRSeverity = "minor"
)

// ProblemReport is a single DO-178C problem report entry.
//
//fusa:req REQ-FO-PR001
type ProblemReport struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	PhaseFound  Phase      `json:"phaseFound"`
	PhaseFixed  Phase      `json:"phaseFixed,omitempty"`
	Severity    PRSeverity `json:"severity"`
	Status      Status     `json:"status"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
	Resolution  string     `json:"resolution,omitempty"`
}

// Log is the complete problem report log for a project.
//
//fusa:req REQ-FO-PR001
type Log struct {
	Project string          `json:"project"`
	Reports []ProblemReport `json:"reports"`
}

// Load reads the problem report log from projectRoot.
// If the file does not exist it returns an empty Log with no error.
//
//fusa:req REQ-FO-PR002
func Load(projectRoot string) (*Log, error) {
	path := filepath.Join(projectRoot, ProblemsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Log{}, nil
		}
		return nil, fmt.Errorf("pr: read %s: %w", ProblemsFile, err)
	}
	var log Log
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("pr: parse %s: %w", ProblemsFile, err)
	}
	return &log, nil
}

// Save writes log to .fusaops-problems.json in projectRoot as indented JSON.
//
//fusa:req REQ-FO-PR002
func Save(projectRoot string, log *Log) error {
	path := filepath.Join(projectRoot, ProblemsFile)
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("pr: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("pr: write %s: %w", ProblemsFile, err)
	}
	return nil
}

// Add appends report to log and returns the updated log.
// The caller is responsible for saving.
//
//fusa:req REQ-FO-PR003
func Add(log *Log, report ProblemReport) *Log {
	now := time.Now().UTC()
	if report.Created.IsZero() {
		report.Created = now
	}
	report.Updated = now
	if report.Status == "" {
		report.Status = StatusOpen
	}
	if report.Severity == "" {
		report.Severity = PRSeverityMinor
	}
	log.Reports = append(log.Reports, report)
	return log
}

// Close marks the report with the given id as closed and records the
// resolution. The caller is responsible for saving.
//
//fusa:req REQ-FO-PR003
func Close(log *Log, id, resolution string) error {
	for i := range log.Reports {
		if log.Reports[i].ID == id {
			log.Reports[i].Status = StatusClosed
			log.Reports[i].Resolution = resolution
			log.Reports[i].Updated = time.Now().UTC()
			return nil
		}
	}
	return fmt.Errorf("pr: report %q not found", id)
}

// Find returns the first report with the given id, or nil if not found.
//
//fusa:req REQ-FO-PR003
func Find(log *Log, id string) *ProblemReport {
	for i := range log.Reports {
		if log.Reports[i].ID == id {
			return &log.Reports[i]
		}
	}
	return nil
}

// Render writes a text or JSON summary of log to w.
//
//fusa:req REQ-FO-PR004
func Render(w io.Writer, log *Log, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(log)
	case "text":
		return renderText(w, log)
	default:
		return fmt.Errorf("pr: unsupported format %q", format)
	}
}

func renderText(w io.Writer, log *Log) error {
	open, closed := 0, 0
	for _, r := range log.Reports {
		if r.Status == StatusClosed {
			closed++
		} else {
			open++
		}
	}
	fmt.Fprintf(w, "Problem Reports: %d total  %d open  %d closed\n\n", len(log.Reports), open, closed)
	for _, r := range log.Reports {
		desc := r.Description
		if desc == "" {
			desc = "(no description)"
		}
		res := ""
		if r.Resolution != "" {
			res = fmt.Sprintf("\n  Resolution: %s", r.Resolution)
		}
		fmt.Fprintf(w, "[%s] %s  (%s / %s)\n  %s\n  Phase: %s  Severity: %s  Status: %s%s\n\n",
			r.ID, r.Title, r.Severity, r.Status,
			desc, r.PhaseFound, r.Severity, r.Status, res)
	}
	return nil
}
