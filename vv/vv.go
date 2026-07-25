// Package vv manages V&V (verification and validation) independence declarations
// for FuSaOps projects.
//
// Independence declarations record the implementationAuthor, independentReviewer,
// and independentTestExecutor for a project, and derive the maximum achievable
// ASIL per ISO 26262-2:2018 §6.4 Table 2.
package vv

import (
	"encoding/json"
	"fmt"
	"io"
)

// VandVFile is the conventional evidence reference name for V&V declarations.
// Declarations are stored in .fusaops.json (vv section), not a separate file;
// this constant is used in auditpack evidence references.
//
//fusa:req REQ-FO-VV001
const VandVFile = ".fusaops-vv.json"

// Declaration holds per-repo V&V independence declarations.
//
//fusa:req REQ-FO-VV001
type Declaration struct {
	// Project is the project name, typically sourced from cfg.Project.Name.
	Project string `json:"project,omitempty"`
	// ImplementationAuthor is the person or team that wrote the implementation.
	ImplementationAuthor string `json:"implementationAuthor,omitempty"`
	// IndependentReviewer is the person (distinct from the author) who performed
	// independent design review, satisfying ISO 26262-2:2018 independence for ASIL-C.
	IndependentReviewer string `json:"independentReviewer,omitempty"`
	// IndependentTestExecutor is the person (distinct from the author) who executed
	// tests independently, satisfying the additional independence for ASIL-D.
	IndependentTestExecutor string `json:"independentTestExecutor,omitempty"`
}

// IndependenceLevel returns the independence level derived from d:
//
//	0 — no independent reviewer or test executor declared
//	1 — independent reviewer declared (no test executor)
//	2 — both independent reviewer and independent test executor declared
//
//fusa:req REQ-FO-VV002
func IndependenceLevel(d Declaration) int {
	switch {
	case d.IndependentReviewer != "" && d.IndependentTestExecutor != "":
		return 2
	case d.IndependentReviewer != "":
		return 1
	default:
		return 0
	}
}

// AchievableASIL returns the maximum ISO 26262 ASIL achievable given the
// independence level of d, per ISO 26262-2:2018 §6.4 Table 2:
//
//	level 0 → "ASIL-B"  (no independence required for ASIL-A/B)
//	level 1 → "ASIL-C"  (independent reviewer satisfies ASIL-C)
//	level 2 → "ASIL-D"  (reviewer + independent test executor satisfies ASIL-D)
//
//fusa:req REQ-FO-VV002
func AchievableASIL(d Declaration) string {
	switch IndependenceLevel(d) {
	case 2:
		return "ASIL-D"
	case 1:
		return "ASIL-C"
	default:
		return "ASIL-B"
	}
}

// Validate returns human-readable issue strings for any consistency problems
// found in d. An empty slice means the declaration is consistent.
//
//fusa:req REQ-FO-VV003
func Validate(d Declaration) []string {
	var issues []string
	if d.ImplementationAuthor == "" {
		issues = append(issues, "implementationAuthor is empty: declarations are unanchored")
	}
	if d.IndependentReviewer != "" && d.ImplementationAuthor != "" &&
		d.IndependentReviewer == d.ImplementationAuthor {
		issues = append(issues, "independentReviewer is the same person as implementationAuthor: not truly independent")
	}
	if d.IndependentTestExecutor != "" && d.ImplementationAuthor != "" &&
		d.IndependentTestExecutor == d.ImplementationAuthor {
		issues = append(issues, "independentTestExecutor is the same person as implementationAuthor: not truly independent")
	}
	if d.IndependentTestExecutor != "" && d.IndependentReviewer == "" {
		issues = append(issues, "independentTestExecutor is set but independentReviewer is empty: ISO 26262 ASIL-C independence is a prerequisite for ASIL-D")
	}
	return issues
}

// renderJSON is the JSON payload emitted by Render for format "json".
type renderJSON struct {
	Project                 string `json:"project,omitempty"`
	ImplementationAuthor    string `json:"implementationAuthor,omitempty"`
	IndependentReviewer     string `json:"independentReviewer,omitempty"`
	IndependentTestExecutor string `json:"independentTestExecutor,omitempty"`
	IndependenceLevel       int    `json:"independenceLevel"`
	AchievableASIL          string `json:"achievableAsil"`
}

// Render writes d plus the derived independence level and achievable ASIL to w.
// Supported formats: "text" (markdown-style table) and "json".
//
//fusa:req REQ-FO-VV004
func Render(w io.Writer, d Declaration, format string) error {
	switch format {
	case "", "text":
		return renderText(w, d)
	case "json":
		return renderJSONFmt(w, d)
	default:
		return fmt.Errorf("vv: unsupported format %q (text, json)", format)
	}
}

func renderText(w io.Writer, d Declaration) error {
	level := IndependenceLevel(d)
	asil := AchievableASIL(d)

	project := d.Project
	if project == "" {
		project = "(unset)"
	}
	author := d.ImplementationAuthor
	if author == "" {
		author = "(unset)"
	}
	reviewer := d.IndependentReviewer
	if reviewer == "" {
		reviewer = "(unset)"
	}
	executor := d.IndependentTestExecutor
	if executor == "" {
		executor = "(unset)"
	}

	_, err := fmt.Fprintf(w,
		"V&V Independence Declaration\n"+
			"============================\n"+
			"Project                    : %s\n"+
			"Implementation Author      : %s\n"+
			"Independent Reviewer       : %s\n"+
			"Independent Test Executor  : %s\n"+
			"Independence Level         : %d\n"+
			"Achievable ASIL            : %s\n",
		project, author, reviewer, executor, level, asil)
	return err
}

func renderJSONFmt(w io.Writer, d Declaration) error {
	payload := renderJSON{
		Project:                 d.Project,
		ImplementationAuthor:    d.ImplementationAuthor,
		IndependentReviewer:     d.IndependentReviewer,
		IndependentTestExecutor: d.IndependentTestExecutor,
		IndependenceLevel:       IndependenceLevel(d),
		AchievableASIL:          AchievableASIL(d),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
