// Package adapter integrates the per-language x-FuSa toolchain into FuSaOps.
//
// Each Adapter wraps one external tool (go-FuSa, c-FuSa, cpp-FuSa, ...). An
// adapter knows how to detect whether its language is present in a project,
// whether its tool binary is installed, and how to run the tool's machine
// readable check and normalise the output into FuSaOps findings.
//
// All current x-FuSa tools emit a common JSON report schema from
// "<tool> check --format json"; parseToolReport decodes it. Adapters are
// registered with the package-level Default registry via init functions in
// gofusa.go, cfusa.go and cpfusa.go.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Adapter wraps a single language-specific x-FuSa tool.
//
//fusa:req REQ-FO-ADP001
type Adapter interface {
	// Name is the human-readable tool name, e.g. "go-FuSa".
	Name() string
	// Language is the programming language this adapter handles.
	Language() fusaops.Language
	// Tool is the executable name expected on PATH, e.g. "gofusa".
	Tool() string
	// Detect reports whether the adapter's language is present under root.
	Detect(root string) (bool, error)
	// Available reports whether the tool binary is installed on PATH.
	Available() bool
	// Check runs the tool against root and returns normalised findings.
	Check(ctx context.Context, root string) ([]fusaops.Finding, error)
}

// runnerFunc executes a command and returns its combined stdout. It is a field
// on cmdAdapter so tests can inject a fake without a real binary on PATH.
type runnerFunc func(ctx context.Context, dir, name string, args ...string) ([]byte, error)

// cmdAdapter is the generic Adapter implementation shared by every x-FuSa tool.
// Concrete adapters differ only in the configured metadata.
//
//fusa:req REQ-FO-ADP002
type cmdAdapter struct {
	name       string
	language   fusaops.Language
	tool       string
	extensions []string // source file extensions that mark the language
	run        runnerFunc
}

func (a *cmdAdapter) Name() string               { return a.name }
func (a *cmdAdapter) Language() fusaops.Language { return a.language }
func (a *cmdAdapter) Tool() string               { return a.tool }

// Available reports whether the tool binary resolves on PATH.
//
//fusa:req REQ-FO-ADP003
func (a *cmdAdapter) Available() bool {
	_, err := exec.LookPath(a.tool)
	return err == nil
}

// Detect walks root looking for any source file matching the adapter's
// language extensions, skipping VCS and common build/vendor directories.
//
//fusa:req REQ-FO-ADP004
func (a *cmdAdapter) Detect(root string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		for _, e := range a.extensions {
			if ext == e {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("adapter %s: detect: %w", a.name, err)
	}
	return found, nil
}

// Check runs "<tool> check --format json --output <tmp>" against root and
// returns normalised findings tagged with this adapter's language and tool.
//
// A non-zero exit status from the tool (which the x-FuSa tools use to signal
// ERROR-severity findings) is not treated as a failure: the report file is
// parsed regardless. Exit 3 (§2.3 runtime/internal error) is different: the
// tool could not complete analysis, so Check returns an error (surfacing the
// report's §3.2 error.message when present) even if a partial findings list
// was decoded — the caller (orchestrator) already records a Check error as a
// skipped component, the same treatment an uninstalled tool gets.
//
//fusa:req REQ-FO-ADP005
func (a *cmdAdapter) Check(ctx context.Context, root string) ([]fusaops.Finding, error) {
	tmp, err := os.CreateTemp("", "fusaops-"+a.tool+"-*.json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: temp file: %w", a.name, err)
	}
	out := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(out) }()

	_, runErr := a.run(ctx, root, a.tool, "check", "--format", "json", "--output", out)

	data, readErr := os.ReadFile(out)
	if readErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("adapter %s: %w", a.name, runErr)
		}
		return nil, fmt.Errorf("adapter %s: read report: %w", a.name, readErr)
	}
	findings, tr, err := parseToolReport(data, a.language, a.name)
	if err != nil {
		return nil, fmt.Errorf("adapter %s: %w", a.name, err)
	}
	if runErr != nil {
		if tr.Error != nil {
			return findings, fmt.Errorf("adapter %s: runtime error (%s): %s", a.name, tr.Error.Code, tr.Error.Message)
		}
		return findings, fmt.Errorf("adapter %s: %w", a.name, runErr)
	}
	if tr.Error != nil {
		return findings, fmt.Errorf("adapter %s: runtime error (%s): %s", a.name, tr.Error.Code, tr.Error.Message)
	}
	return findings, nil
}

// defaultRunner executes the command in dir and returns combined output.
// A non-zero exit status is swallowed (treated as "checks found issues", the
// normal §2.3 exit-1 gate-failure case) EXCEPT exit 3, which is a §2.3
// runtime/internal error — the tool could not complete analysis, so it is
// surfaced as a real error rather than silently treated as a clean run.
func defaultRunner(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 3 {
				return out, fmt.Errorf("%s exited 3 (runtime/internal error): %w", name, exitErr)
			}
			return out, nil // non-zero exit is expected when findings exist
		}
		return out, err
	}
	return out, nil
}

// toolReport mirrors the common JSON schema emitted by every x-FuSa tool's
// "check --format json". Only the fields FuSaOps needs are decoded.
type toolReport struct {
	Findings []toolFinding `json:"findings"`
	Error    *toolError    `json:"error,omitempty"`
}

// toolError mirrors the §3.2 structured runtime-error envelope, present only
// when the tool paired a partial document with exit 3.
type toolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type toolFinding struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Location struct {
		File      string `json:"file"`
		Line      int    `json:"line"`
		Column    int    `json:"column"`
		EndLine   int    `json:"endLine"`
		EndColumn int    `json:"endColumn"`
	} `json:"location"`
	Category    string `json:"category"`
	Disposition string `json:"disposition"`
	Standard    string `json:"standard"`
	Clause      string `json:"clause"`
	Remediation string `json:"remediation"`
	Fingerprint string `json:"fingerprint"`
}

// parseToolReport decodes a tool's JSON report into FuSaOps findings, tagging
// each with the originating language and tool name. It also returns the
// decoded toolReport itself so the caller can inspect the §3.2 error field.
//
//fusa:req REQ-FO-ADP006
func parseToolReport(data []byte, lang fusaops.Language, tool string) ([]fusaops.Finding, toolReport, error) {
	var tr toolReport
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, tr, fmt.Errorf("parse tool report: %w", err)
	}
	findings := make([]fusaops.Finding, 0, len(tr.Findings))
	for _, f := range tr.Findings {
		findings = append(findings, fusaops.Finding{
			Language:    lang,
			Tool:        tool,
			RuleID:      f.RuleID,
			Severity:    normaliseSeverity(f.Severity),
			Message:     f.Message,
			Location:    fusaops.Location{File: f.Location.File, Line: f.Location.Line, Column: f.Location.Column, EndLine: f.Location.EndLine, EndColumn: f.Location.EndColumn},
			Category:    normaliseCategory(f.Category),
			Disposition: f.Disposition,
			Standard:    f.Standard,
			Clause:      f.Clause,
			Remediation: f.Remediation,
			Fingerprint: f.Fingerprint,
		})
	}
	return findings, tr, nil
}

// normaliseSeverity maps a tool's severity string onto a FuSaOps Severity,
// defaulting unknown values to INFO so a misbehaving tool cannot silently
// downgrade a real problem to nothing.
func normaliseSeverity(s string) fusaops.Severity {
	switch fusaops.Severity(s) {
	case fusaops.SeverityError:
		return fusaops.SeverityError
	case fusaops.SeverityWarning:
		return fusaops.SeverityWarning
	default:
		return fusaops.SeverityInfo
	}
}

// validCategories is the x-FuSa spec §4 closed, extensible category enum.
var validCategories = map[string]bool{
	"lint": true, "style": true, "safety": true, "security": true,
	"coverage": true, "requirement": true, "concurrency": true,
	"supply-chain": true, "config": true, "other": true,
}

// normaliseCategory maps any category value outside the §4 closed enum to
// "other", per the spec's consumer MUST — mirroring normaliseSeverity's
// fail-safe treatment of an unrecognised value.
func normaliseCategory(c string) string {
	if validCategories[c] {
		return c
	}
	return "other"
}

// DefaultSkipDirs is the set of directory names skipped during language
// detection. It is a package-level variable rather than a hard-coded switch so
// a caller can tailor the skip-list (e.g. remove "out"/"dist"/"target" when a
// project legitimately keeps sources there, or add project-specific excludes)
// before running detection.
//
//fusa:req REQ-FO-ADP004
var DefaultSkipDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, "node_modules": true,
	"vendor": true, "build": true, "dist": true, ".cache": true,
	"target": true, "out": true, ".idea": true, ".vscode": true,
}

// skipDir reports whether a directory should be skipped during detection.
func skipDir(name string) bool {
	return DefaultSkipDirs[name]
}

// Registry holds a set of registered adapters keyed by tool name.
//
//fusa:req REQ-FO-ADP007
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry returns an empty Registry.
//
//fusa:req REQ-FO-ADP030
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds a to the registry, returning an error on nil or duplicate tool.
//
//fusa:req REQ-FO-ADP030
func (r *Registry) Register(a Adapter) error {
	if a == nil {
		return fmt.Errorf("adapter: cannot register nil adapter")
	}
	if _, dup := r.adapters[a.Tool()]; dup {
		return fmt.Errorf("adapter: tool %q already registered", a.Tool())
	}
	r.adapters[a.Tool()] = a
	return nil
}

// MustRegister calls Register and panics on error. For use in init functions.
//
//fusa:req REQ-FO-ADP030
func (r *Registry) MustRegister(a Adapter) {
	if err := r.Register(a); err != nil {
		panic(err)
	}
}

// All returns every registered adapter sorted by tool name.
//
//fusa:req REQ-FO-ADP030
func (r *Registry) All() []Adapter {
	out := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tool() < out[j].Tool() })
	return out
}

// Applicable returns the adapters whose language is detected under root.
//
//fusa:req REQ-FO-ADP008
func (r *Registry) Applicable(root string) ([]Adapter, error) {
	var out []Adapter
	for _, a := range r.All() {
		ok, err := a.Detect(root)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// Default is the package-level registry populated by the built-in adapters.
//
//fusa:req REQ-FO-ADP009
var Default = NewRegistry()
