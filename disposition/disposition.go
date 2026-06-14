// Package disposition manages finding disposition entries for FuSaOps projects.
//
// A disposition entry records that a specific finding (by rule ID and language)
// has been reviewed and accepted or scheduled for fixing. This provides an audit
// trail distinct from suppression: suppressed findings are hidden; dispositioned
// findings are acknowledged and tracked.
//
// Usage:
//
//	log, err := disposition.Load(projectRoot)
//	log = disposition.Add(log, disposition.Entry{...})
//	err = disposition.Save(projectRoot, log)
package disposition

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// DispositionsFile is the default filename for the dispositions log.
const DispositionsFile = ".fusaops-dispositions.json"

// Action describes what was decided for a finding.
//
//fusa:req REQ-FO-DISP001
type Action string

const (
	// ActionAccept records the finding as accepted/waived with rationale.
	ActionAccept Action = "accept"
	// ActionFix records the finding as scheduled for remediation.
	ActionFix Action = "fix"
)

// Entry records a single disposition decision for a finding.
//
//fusa:req REQ-FO-DISP001
type Entry struct {
	RuleID    string    `json:"ruleID"`
	Language  string    `json:"language,omitempty"`
	Rationale string    `json:"rationale"`
	Reviewer  string    `json:"reviewer"`
	Date      time.Time `json:"date"`
	Action    Action    `json:"action"`
	Reference string    `json:"reference,omitempty"`
}

// Log is the full dispositions log for a project.
//
//fusa:req REQ-FO-DISP001
type Log struct {
	Project string  `json:"project"`
	Entries []Entry `json:"entries"`
}

// Load reads the dispositions log from projectRoot.
// If the file does not exist it returns an empty log with no error.
//
//fusa:req REQ-FO-DISP002
func Load(projectRoot string) (*Log, error) {
	path := filepath.Join(projectRoot, DispositionsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Log{}, nil
		}
		return nil, fmt.Errorf("disposition: read %s: %w", DispositionsFile, err)
	}
	var log Log
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("disposition: parse %s: %w", DispositionsFile, err)
	}
	return &log, nil
}

// Save writes the dispositions log to .fusaops-dispositions.json in projectRoot.
//
//fusa:req REQ-FO-DISP002
func Save(projectRoot string, log *Log) error {
	path := filepath.Join(projectRoot, DispositionsFile)
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("disposition: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("disposition: write %s: %w", DispositionsFile, err)
	}
	return nil
}

// Add appends e to log and returns the updated log.
//
//fusa:req REQ-FO-DISP002
func Add(log *Log, e Entry) *Log {
	log.Entries = append(log.Entries, e)
	return log
}

// Find returns the first entry matching ruleID (and optionally language).
// Returns nil if not found.
//
//fusa:req REQ-FO-DISP002
func Find(log *Log, ruleID, language string) *Entry {
	for i := range log.Entries {
		e := &log.Entries[i]
		if e.RuleID == ruleID && (language == "" || e.Language == language || e.Language == "") {
			return e
		}
	}
	return nil
}

// RenderEntries writes the disposition log entries in text format.
//
//fusa:req REQ-FO-DISP003
func RenderEntries(w io.Writer, log *Log) error {
	if len(log.Entries) == 0 {
		fmt.Fprintln(w, "No disposition entries.")
		return nil
	}
	project := log.Project
	if project == "" {
		project = "(project)"
	}
	fmt.Fprintf(w, "FuSaOps Dispositions — %s (%d entries)\n\n", project, len(log.Entries))
	for _, e := range log.Entries {
		lang := ""
		if e.Language != "" {
			lang = fmt.Sprintf(" [%s]", e.Language)
		}
		ref := ""
		if e.Reference != "" {
			ref = fmt.Sprintf(" (%s)", e.Reference)
		}
		fmt.Fprintf(w, "  [%s]%s %s%s — %s\n    Reviewer: %s  Date: %s\n    %s\n\n",
			e.Action, lang, e.RuleID, ref, e.Rationale,
			e.Reviewer, e.Date.Format("2006-01-02"), e.Rationale)
	}
	return nil
}
