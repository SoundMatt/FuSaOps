// Package verify provides test evidence collection for FuSaOps projects.
//
// Use Run to execute the Go test suite and capture structured results, then New
// to build an evidence bundle and Save to persist it. The bundle provides an
// auditable record of test execution for safety evidence packages.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// BundleFile is the default filename for the evidence bundle.
//
//fusa:req REQ-FO-VER001
const BundleFile = ".fusaops-evidence.json"

// TestStatus is the outcome of a single test run.
//
//fusa:req REQ-FO-VER001
type TestStatus string

const (
	StatusPass TestStatus = "pass"
	StatusFail TestStatus = "fail"
	StatusSkip TestStatus = "skip"
)

// TestResult holds the result of a single test function.
//
//fusa:req REQ-FO-VER001
type TestResult struct {
	Name    string     `json:"name"`
	Package string     `json:"package"`
	Status  TestStatus `json:"status"`
	Elapsed float64    `json:"elapsedSeconds"`
}

// Summary holds aggregate test result counts.
//
//fusa:req REQ-FO-VER001
type Summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Bundle is the verification evidence bundle persisted to BundleFile.
//
//fusa:req REQ-FO-VER001
type Bundle struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	ProjectRoot string       `json:"projectRoot"`
	GoVersion   string       `json:"goVersion"`
	Results     []TestResult `json:"results"`
	Summary     Summary      `json:"summary"`
}

// testEvent is one line of go test -json output.
type testEvent struct {
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Package string  `json:"Package"`
	Elapsed float64 `json:"Elapsed"`
}

// Parse reads go test -json output from r and returns per-test results.
// Package-level events (no Test field) are ignored.
//
//fusa:req REQ-FO-VER002
func Parse(r io.Reader) ([]TestResult, error) {
	dec := json.NewDecoder(r)
	var results []TestResult
	for {
		var ev testEvent
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("verify: parse: %w", err)
		}
		if ev.Test == "" {
			continue // package-level event
		}
		switch ev.Action {
		case "pass":
			results = append(results, TestResult{Name: ev.Test, Package: ev.Package, Status: StatusPass, Elapsed: ev.Elapsed})
		case "fail":
			results = append(results, TestResult{Name: ev.Test, Package: ev.Package, Status: StatusFail, Elapsed: ev.Elapsed})
		case "skip":
			results = append(results, TestResult{Name: ev.Test, Package: ev.Package, Status: StatusSkip, Elapsed: ev.Elapsed})
		}
	}
	return results, nil
}

// Summarise computes aggregate counts from a slice of TestResults.
//
//fusa:req REQ-FO-VER002
func Summarise(results []TestResult) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
		case StatusSkip:
			s.Skipped++
		}
	}
	return s
}

// Run executes go test -json -count=1 ./... in dir and returns parsed results.
// A test-failure exit code is not an error; results will contain StatusFail
// entries. Other execution errors (go not found, no module) are returned as errors.
//
//fusa:req REQ-FO-VER003
func Run(ctx context.Context, dir string) ([]TestResult, error) {
	cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", "./...")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, fmt.Errorf("verify: run: %w", err)
		}
		// Non-zero exit means test failures; still parse whatever was written.
	}
	return Parse(bytes.NewReader(out))
}

// New builds a Bundle from test results for the given project root.
//
//fusa:req REQ-FO-VER004
func New(projectRoot string, results []TestResult) *Bundle {
	return &Bundle{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: projectRoot,
		GoVersion:   runtime.Version(),
		Results:     results,
		Summary:     Summarise(results),
	}
}

// Save writes the evidence bundle to path as indented JSON.
//
//fusa:req REQ-FO-VER004
func Save(path string, b *Bundle) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("verify: marshal bundle: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("verify: write %s: %w", path, err)
	}
	return nil
}

// Load reads an evidence bundle from path.
//
//fusa:req REQ-FO-VER004
func Load(path string) (*Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("verify: read %s: %w", path, err)
	}
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("verify: unmarshal %s: %w", path, err)
	}
	return &b, nil
}

// Render writes a representation of the bundle to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-VER005
func Render(w io.Writer, b *Bundle, format string) error {
	switch format {
	case "text", "":
		return renderText(w, b)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(b)
	default:
		return fmt.Errorf("verify: unsupported format %q", format)
	}
}

func renderText(w io.Writer, b *Bundle) error {
	s := b.Summary
	fmt.Fprintf(w, "Project:   %s\n", b.ProjectRoot)
	fmt.Fprintf(w, "Go:        %s\n", b.GoVersion)
	fmt.Fprintf(w, "Generated: %s\n", b.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Tests:     %d total  %d passed  %d failed  %d skipped\n",
		s.Total, s.Passed, s.Failed, s.Skipped)
	if s.Failed > 0 {
		fmt.Fprintf(w, "\nFailed tests:\n")
		for _, r := range b.Results {
			if r.Status == StatusFail {
				fmt.Fprintf(w, "  FAIL  %s  %s\n", r.Package, r.Name)
			}
		}
	}
	return nil
}
