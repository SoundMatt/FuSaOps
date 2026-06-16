// Package qualify aggregates per-tool qualification reports from x-FuSa
// adapters into a cross-language tool confidence summary.
//
// Each x-FuSa tool that supports the "qualify" subcommand emits a qualification
// report via the adapter.Qualifier interface. FuSaOps collects these into a
// single Report suitable as tool confidence evidence in regulated environments
// (ISO 26262 Part 8 §11, IEC 61508 Part 6 §7.4).
package qualify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
)

// ReportFile is the default filename for the qualification report.
//
//fusa:req REQ-FO-QUAL001
const ReportFile = ".fusaops-qualify-report.json"

// ComponentResult holds one adapter's qualification outcome.
//
//fusa:req REQ-FO-QUAL001
type ComponentResult struct {
	Language  string `json:"language"`
	Tool      string `json:"tool"`
	Available bool   `json:"available"`
	Skipped   string `json:"skipped,omitempty"`
	Total     int    `json:"total"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
}

// Passed reports whether the component qualification passed (no failures).
func (c ComponentResult) AllPassed() bool { return c.Skipped == "" && c.Failed == 0 }

// Report is the cross-language qualification roll-up.
//
//fusa:req REQ-FO-QUAL001
type Report struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	GoVersion   string            `json:"goVersion"`
	ProjectRoot string            `json:"projectRoot"`
	Total       int               `json:"total"`
	Passed      int               `json:"passed"`
	Failed      int               `json:"failed"`
	Components  []ComponentResult `json:"components"`
	Hash        string            `json:"hash"`
}

// HasFailures reports whether any component failed qualification.
func (r *Report) HasFailures() bool { return r.Failed > 0 }

// Run collects qualification results from all applicable adapters under root.
// Adapters that do not implement adapter.Qualifier, are unavailable, or whose
// qualify call fails are recorded as skipped rather than fatal.
//
//fusa:req REQ-FO-QUAL002
func Run(ctx context.Context, adapters []adapter.Adapter, root string) (*Report, error) {
	report := &Report{
		GeneratedAt: time.Now().UTC(),
		GoVersion:   runtime.Version(),
		ProjectRoot: root,
		Components:  make([]ComponentResult, len(adapters)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, a := range adapters {
		wg.Add(1)
		go func(i int, a adapter.Adapter) {
			defer wg.Done()
			cr := ComponentResult{
				Language:  a.Language().String(),
				Tool:      a.Tool(),
				Available: a.Available(),
			}
			if !a.Available() {
				cr.Skipped = "tool not installed"
				mu.Lock()
				report.Components[i] = cr
				mu.Unlock()
				return
			}
			q, ok := a.(adapter.Qualifier)
			if !ok {
				cr.Skipped = "adapter does not support qualify"
				mu.Lock()
				report.Components[i] = cr
				mu.Unlock()
				return
			}
			qr, err := q.Qualify(ctx, root)
			if err != nil {
				cr.Skipped = err.Error()
				mu.Lock()
				report.Components[i] = cr
				mu.Unlock()
				return
			}
			cr.Total = qr.Total
			cr.Passed = qr.Passed
			cr.Failed = qr.Failed
			mu.Lock()
			report.Components[i] = cr
			report.Total += qr.Total
			report.Passed += qr.Passed
			report.Failed += qr.Failed
			mu.Unlock()
		}(i, a)
	}
	wg.Wait()

	report.Hash = computeHash(report)
	return report, nil
}

// Save writes the report to path as indented JSON.
//
//fusa:req REQ-FO-QUAL003
func Save(path string, r *Report) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("qualify: marshal report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("qualify: write %s: %w", path, err)
	}
	return nil
}

// Load reads a qualification report from path.
//
//fusa:req REQ-FO-QUAL003
func Load(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("qualify: read %s: %w", path, err)
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("qualify: unmarshal %s: %w", path, err)
	}
	return &r, nil
}

// Render writes the report to w in the given format ("text" or "json").
//
//fusa:req REQ-FO-QUAL004
func Render(w io.Writer, r *Report, format string) error {
	switch format {
	case "text", "":
		return renderText(w, r)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	default:
		return fmt.Errorf("qualify: unsupported format %q", format)
	}
}

func renderText(w io.Writer, r *Report) error {
	fmt.Fprintf(w, "Project:   %s\n", r.ProjectRoot)
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Overall:   %d total  %d passed  %d failed\n", r.Total, r.Passed, r.Failed)
	fmt.Fprintf(w, "\nComponents:\n")
	for _, c := range r.Components {
		if c.Skipped != "" {
			fmt.Fprintf(w, "  %-10s %-12s  skipped (%s)\n", c.Language, c.Tool, c.Skipped)
		} else {
			status := "PASS"
			if c.Failed > 0 {
				status = "FAIL"
			}
			fmt.Fprintf(w, "  %-10s %-12s  %s  %d/%d passed\n", c.Language, c.Tool, status, c.Passed, c.Total)
		}
	}
	return nil
}

func computeHash(r *Report) string {
	data, _ := json.Marshal(struct {
		GeneratedAt time.Time         `json:"generatedAt"`
		Total       int               `json:"total"`
		Passed      int               `json:"passed"`
		Failed      int               `json:"failed"`
		Components  []ComponentResult `json:"components"`
	}{r.GeneratedAt, r.Total, r.Passed, r.Failed, r.Components})
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h)
}
