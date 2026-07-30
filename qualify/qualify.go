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
	"sort"
	"sync"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
)

// ReportFile is the default filename for the qualification report.
//
//fusa:req REQ-FO-QUAL001
const ReportFile = ".fusaops-qualify-report.json"

// QualificationType distinguishes how a tool-qualification run was performed.
//
//fusa:req REQ-FO-QUAL005
type QualificationType string

const (
	// QualificationTypeSelf is the default: the project team ran qualification
	// against its own x-FuSa tools.
	QualificationTypeSelf QualificationType = "self"
	// QualificationTypeIndependent signals a TQL-5 / DO-330 externally
	// certified qualification; a RecordUri pointing to the certificate is
	// expected alongside this type.
	QualificationTypeIndependent QualificationType = "independent"
)

// RunOptions configures optional metadata for a qualification run.
//
//fusa:req REQ-FO-QUAL005
type RunOptions struct {
	// Type identifies the qualification approach. Empty defaults to
	// QualificationTypeSelf.
	Type QualificationType
	// RecordUri is a URI pointing to the external qualification certificate
	// (e.g. a TQL-5 or DO-330 record). Only meaningful when Type is
	// QualificationTypeIndependent.
	RecordUri string
}

// ComponentResult holds one adapter's qualification outcome.
//
//fusa:req REQ-FO-QUAL001
//fusa:req REQ-FO-QLF010
type ComponentResult struct {
	Language  string `json:"language"`
	Tool      string `json:"tool"`
	Available bool   `json:"available"`
	Skipped   string `json:"skipped,omitempty"`
	Total     int    `json:"total"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
	// V&V independence fields (REQ-FO-QLF010).
	QualificationMethod    string `json:"qualificationMethod,omitempty"`
	QualifierIdentity      string `json:"qualifierIdentity,omitempty"`
	QualificationRecordUri string `json:"qualificationRecordUri,omitempty"`
	ImplementationAuthor   string `json:"implementationAuthor,omitempty"`
	IndependentReviewer    string `json:"independentReviewer,omitempty"`
	AchievableASIL         string `json:"achievableAsil,omitempty"`
}

// AllPassed reports whether the component qualification passed (no failures).
//
//fusa:req REQ-FO-QUAL008
func (c ComponentResult) AllPassed() bool { return c.Skipped == "" && c.Failed == 0 }

// IsIndependent reports whether this component used an independent reviewer,
// indicating a higher trust level for the qualification evidence.
//
//fusa:req REQ-FO-QLF011
func (c ComponentResult) IsIndependent() bool { return c.IndependentReviewer != "" }

// Report is the cross-language qualification roll-up.
//
//fusa:req REQ-FO-QUAL001
//fusa:req REQ-FO-QLF010
type Report struct {
	GeneratedAt            time.Time `json:"generatedAt"`
	GoVersion              string    `json:"goVersion"`
	ProjectRoot            string    `json:"projectRoot"`
	QualificationType      string    `json:"qualificationType,omitempty"`      // REQ-FO-QUAL005
	QualificationRecordUri string    `json:"qualificationRecordUri,omitempty"` // REQ-FO-QUAL005
	// V&V independence fields (REQ-FO-QLF010).
	QualificationMethod  string            `json:"qualificationMethod,omitempty"`
	QualifierIdentity    string            `json:"qualifierIdentity,omitempty"`
	ImplementationAuthor string            `json:"implementationAuthor,omitempty"`
	IndependentReviewer  string            `json:"independentReviewer,omitempty"`
	AchievableASIL       string            `json:"achievableAsil,omitempty"`
	Total                int               `json:"total"`
	Passed               int               `json:"passed"`
	Failed               int               `json:"failed"`
	Components           []ComponentResult `json:"components"`
	Hash                 string            `json:"hash"`
}

// IsIndependent reports whether the qualification was performed with an
// independent reviewer (higher trust level per ISO 26262 Part 8 §11).
//
//fusa:req REQ-FO-QLF011
func (r *Report) IsIndependent() bool { return r.IndependentReviewer != "" }

// HasFailures reports whether any component failed qualification.
//
//fusa:req REQ-FO-QUAL008
func (r *Report) HasFailures() bool { return r.Failed > 0 }

// Run collects qualification results from all applicable adapters under root.
// Adapters that do not implement adapter.Qualifier, are unavailable, or whose
// qualify call fails are recorded as skipped rather than fatal.
//
//fusa:req REQ-FO-QUAL002
//fusa:req REQ-FO-QUAL006
func Run(ctx context.Context, adapters []adapter.Adapter, root string, opts ...RunOptions) (*Report, error) {
	report := &Report{
		GeneratedAt: time.Now().UTC(),
		GoVersion:   runtime.Version(),
		ProjectRoot: root,
		Components:  make([]ComponentResult, len(adapters)),
	}

	// Apply RunOptions: default type is "self".
	report.QualificationType = string(QualificationTypeSelf)
	if len(opts) > 0 {
		if opts[0].Type != "" {
			report.QualificationType = string(opts[0].Type)
		}
		report.QualificationRecordUri = opts[0].RecordUri
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
			cr.IndependentReviewer = qr.IndependentReviewer
			cr.QualificationMethod = qr.QualificationMethod
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

// renderText writes a human-readable text summary of the qualification report.
//
//fusa:req REQ-FO-QUAL007
//fusa:req REQ-FO-QLF010
func renderText(w io.Writer, r *Report) error {
	fmt.Fprintf(w, "Project:   %s\n", r.ProjectRoot)
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	if r.QualificationType != "" {
		fmt.Fprintf(w, "Type:      %s\n", r.QualificationType)
	}
	if r.QualificationRecordUri != "" {
		fmt.Fprintf(w, "Record:    %s\n", r.QualificationRecordUri)
	}
	if r.QualificationMethod != "" {
		fmt.Fprintf(w, "Method:    %s\n", r.QualificationMethod)
	}
	if r.QualifierIdentity != "" {
		fmt.Fprintf(w, "Qualifier: %s\n", r.QualifierIdentity)
	}
	if r.ImplementationAuthor != "" {
		fmt.Fprintf(w, "Author:    %s\n", r.ImplementationAuthor)
	}
	if r.IndependentReviewer != "" {
		fmt.Fprintf(w, "Reviewer:  %s (independent)\n", r.IndependentReviewer)
	}
	if r.AchievableASIL != "" {
		fmt.Fprintf(w, "Achievable ASIL: %s\n", r.AchievableASIL)
	}
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
			ind := ""
			if c.IsIndependent() {
				ind = " [independent]"
			}
			fmt.Fprintf(w, "  %-10s %-12s  %s  %d/%d passed%s\n", c.Language, c.Tool, status, c.Passed, c.Total, ind)
		}
	}
	return nil
}

// computeHash canonicalizes the same fields as before (everything except Hash
// itself) via fusaops.Canonicalize, so the hash is genuine RFC 8785 sorted-key
// canonical JSON (§6). Per x-FuSa spec MUST-145/148 the volatile generatedAt
// timestamp is blanked before hashing (otherwise the integrity hash changes on
// every run and cannot anchor tamper/regression detection); per MUST-146 the
// components are hashed in a name-sorted order independent of registry order.
func computeHash(r *Report) string {
	comps := make([]ComponentResult, len(r.Components))
	copy(comps, r.Components)
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].Language != comps[j].Language {
			return comps[i].Language < comps[j].Language
		}
		return comps[i].Tool < comps[j].Tool
	})
	data, _ := json.Marshal(struct {
		GeneratedAt            time.Time         `json:"generatedAt"`
		QualificationType      string            `json:"qualificationType,omitempty"`
		QualificationRecordUri string            `json:"qualificationRecordUri,omitempty"`
		QualificationMethod    string            `json:"qualificationMethod,omitempty"`
		QualifierIdentity      string            `json:"qualifierIdentity,omitempty"`
		ImplementationAuthor   string            `json:"implementationAuthor,omitempty"`
		IndependentReviewer    string            `json:"independentReviewer,omitempty"`
		AchievableASIL         string            `json:"achievableAsil,omitempty"`
		Total                  int               `json:"total"`
		Passed                 int               `json:"passed"`
		Failed                 int               `json:"failed"`
		Components             []ComponentResult `json:"components"`
	}{
		// generatedAt deliberately left as the zero time so the hash is stable.
		time.Time{}, r.QualificationType, r.QualificationRecordUri,
		r.QualificationMethod, r.QualifierIdentity,
		r.ImplementationAuthor, r.IndependentReviewer, r.AchievableASIL,
		r.Total, r.Passed, r.Failed, comps,
	})
	canon, err := fusaops.Canonicalize(data)
	if err != nil {
		canon = data
	}
	h := sha256.Sum256(canon)
	return fmt.Sprintf("sha256:%x", h)
}
