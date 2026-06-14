// Package impact analyses the effect of source-code changes on requirements,
// test evidence, and safety artefacts for multi-language FuSaOps projects.
//
// It runs git diff to discover changed files, greps source annotations
// (//fusa:req, //fusa:test, #fusa:req, #fusa:test) across all supported
// languages to find impacted requirements, and checks whether evidence
// artefacts are stale relative to the changed source files.
package impact

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FileChange describes a single file that changed in the diff.
//
//fusa:req REQ-FO-IMP001
type FileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"` // "M", "A", "D", "R"
}

// RequirementImpact describes a requirement affected by changes.
//
//fusa:req REQ-FO-IMP001
type RequirementImpact struct {
	RequirementID string   `json:"requirementID"`
	AffectedFiles []string `json:"affectedFiles"`
	TestsNeeded   []string `json:"testsNeeded"`
	Stale         bool     `json:"stale"`
}

// ArtifactStatus reports whether a safety artefact is stale.
//
//fusa:req REQ-FO-IMP001
type ArtifactStatus struct {
	File   string `json:"file"`
	Stale  bool   `json:"stale"`
	Reason string `json:"reason,omitempty"`
}

// Report is the complete impact analysis.
//
//fusa:req REQ-FO-IMP001
type Report struct {
	Generated      time.Time           `json:"generated"`
	ChangedFiles   []FileChange        `json:"changedFiles"`
	ImpactedReqs   []RequirementImpact `json:"impactedReqs"`
	StaleArtifacts []ArtifactStatus    `json:"staleArtifacts"`
	RerunTests     []string            `json:"rerunTests"`
}

// evidenceArtifacts are the cross-language safety evidence files to check.
var evidenceArtifacts = []string{
	"check-report.json",
	"aggregate-report.json",
	"coverage-report.json",
	"sbom.json",
	"audit-pack.zip",
	"iso26262-gap-report.json",
	"iec61508-gap-report.json",
	"do178-gap-report.json",
	".fusaops-metrics.json",
}

// sourceExtensions are the file extensions scanned for fusa annotations.
var sourceExtensions = map[string]bool{
	".go": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".rs": true, ".py": true, ".java": true,
}

// Analyse runs a change-impact analysis for projectRoot.
// If fromRef and toRef are both empty, it diffs the working tree against HEAD.
// If only fromRef is set, it diffs fromRef..HEAD. If both are set, fromRef..toRef.
//
//fusa:req REQ-FO-IMP002
func Analyse(projectRoot, fromRef, toRef string) (*Report, error) {
	rep := &Report{Generated: time.Now().UTC()}

	changes, err := changedFiles(projectRoot, fromRef, toRef)
	if err != nil {
		// git unavailable or no repo — still check artifact staleness
		rep.StaleArtifacts = checkArtifacts(projectRoot, time.Time{})
		return rep, nil
	}
	rep.ChangedFiles = changes

	if len(changes) == 0 {
		rep.StaleArtifacts = checkArtifacts(projectRoot, time.Time{})
		return rep, nil
	}

	// Find the latest mtime of changed source files.
	var latestSrc time.Time
	for _, c := range changes {
		abs := filepath.Join(projectRoot, filepath.FromSlash(c.Path))
		if info, statErr := os.Stat(abs); statErr == nil {
			if info.ModTime().After(latestSrc) {
				latestSrc = info.ModTime()
			}
		}
	}

	changedSet := make(map[string]bool)
	for _, c := range changes {
		changedSet[c.Path] = true
		changedSet[filepath.FromSlash(c.Path)] = true
	}

	// Scan annotation maps across all source files.
	reqImpl, reqTest := scanAnnotations(projectRoot)

	// Find impacted requirements.
	rerunSet := make(map[string]bool)
	seen := make(map[string]bool)
	for reqID, implFiles := range reqImpl {
		var affected []string
		for _, f := range implFiles {
			if changedSet[f] || changedSet[filepath.ToSlash(f)] {
				affected = appendUniq(affected, f)
			}
		}
		if len(affected) == 0 || seen[reqID] {
			continue
		}
		seen[reqID] = true
		tests := reqTest[reqID]
		for _, t := range tests {
			rerunSet[t] = true
		}
		rep.ImpactedReqs = append(rep.ImpactedReqs, RequirementImpact{
			RequirementID: reqID,
			AffectedFiles: affected,
			TestsNeeded:   tests,
			Stale:         len(tests) > 0,
		})
	}

	for t := range rerunSet {
		rep.RerunTests = appendUniq(rep.RerunTests, t)
	}

	rep.StaleArtifacts = checkArtifacts(projectRoot, latestSrc)
	return rep, nil
}

// scanAnnotations greps all source files under root for //fusa:req and
// //fusa:test (and #fusa:req / #fusa:test for Python/Rust) annotations.
// Returns two maps: reqID → impl files, reqID → test files.
func scanAnnotations(root string) (reqImpl, reqTest map[string][]string) {
	reqImpl = make(map[string][]string)
	reqTest = make(map[string][]string)

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "vendor" || name == "node_modules" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !sourceExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer func() { _ = f.Close() }()
		scanFile(rel, f, reqImpl, reqTest)
		return nil
	})
	return reqImpl, reqTest
}

func scanFile(rel string, r io.Reader, reqImpl, reqTest map[string][]string) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if id, ok := extractAnnotation(line, "fusa:req"); ok {
			reqImpl[id] = appendUniq(reqImpl[id], rel)
		}
		if id, ok := extractAnnotation(line, "fusa:test"); ok {
			reqTest[id] = appendUniq(reqTest[id], rel)
		}
	}
}

func extractAnnotation(line, keyword string) (string, bool) {
	// Matches: //fusa:req REQ-ID, #fusa:req REQ-ID, *fusa:req REQ-ID
	for _, prefix := range []string{"//", "#", "*"} {
		trimmed := strings.TrimPrefix(line, prefix)
		if strings.HasPrefix(trimmed, keyword+" ") {
			id := strings.TrimSpace(strings.TrimPrefix(trimmed, keyword+" "))
			if id != "" && !strings.Contains(id, " ") {
				return id, true
			}
		}
	}
	return "", false
}

func changedFiles(projectRoot, fromRef, toRef string) ([]FileChange, error) {
	var args []string
	if fromRef == "" && toRef == "" {
		args = []string{"diff", "--name-status", "HEAD"}
	} else if toRef == "" {
		args = []string{"diff", "--name-status", fromRef + "..HEAD"}
	} else {
		args = []string{"diff", "--name-status", fromRef + ".." + toRef}
	}
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = projectRoot
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("impact: git diff: %w", err)
	}
	var changes []FileChange
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		changes = append(changes, FileChange{Path: parts[1], Status: string(parts[0][0])})
	}
	return changes, sc.Err()
}

func checkArtifacts(root string, latestSrc time.Time) []ArtifactStatus {
	var result []ArtifactStatus
	for _, name := range evidenceArtifacts {
		abs := filepath.Join(root, name)
		info, err := os.Stat(abs)
		if err != nil {
			result = append(result, ArtifactStatus{File: name, Stale: true, Reason: "file not present"})
			continue
		}
		if !latestSrc.IsZero() && info.ModTime().Before(latestSrc) {
			result = append(result, ArtifactStatus{
				File:   name,
				Stale:  true,
				Reason: fmt.Sprintf("last updated %s, older than changed sources", info.ModTime().Format("2006-01-02 15:04:05")),
			})
		} else {
			result = append(result, ArtifactStatus{File: name, Stale: false})
		}
	}
	return result
}

func appendUniq(s []string, v string) []string {
	for _, existing := range s {
		if existing == v {
			return s
		}
	}
	return append(s, v)
}

// Render writes the impact report to w in the requested format (text or json).
//
//fusa:req REQ-FO-IMP003
func Render(w io.Writer, rep *Report, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	case "text":
		return renderText(w, rep)
	default:
		return fmt.Errorf("impact: unsupported format %q", format)
	}
}

func renderText(w io.Writer, rep *Report) error {
	fmt.Fprintf(w, "FuSaOps Impact Analysis — %s\n\n", rep.Generated.Format("2006-01-02 15:04:05"))

	fmt.Fprintf(w, "Changed files (%d):\n", len(rep.ChangedFiles))
	if len(rep.ChangedFiles) == 0 {
		fmt.Fprintln(w, "  (no changes detected)")
	}
	for _, c := range rep.ChangedFiles {
		fmt.Fprintf(w, "  [%s] %s\n", c.Status, c.Path)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Impacted requirements (%d):\n", len(rep.ImpactedReqs))
	if len(rep.ImpactedReqs) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, ir := range rep.ImpactedReqs {
		fmt.Fprintf(w, "  %s\n", ir.RequirementID)
		for _, f := range ir.AffectedFiles {
			fmt.Fprintf(w, "    impl: %s\n", f)
		}
		for _, t := range ir.TestsNeeded {
			fmt.Fprintf(w, "    test: %s\n", t)
		}
	}
	fmt.Fprintln(w)

	staleCount := 0
	for _, a := range rep.StaleArtifacts {
		if a.Stale {
			staleCount++
		}
	}
	fmt.Fprintf(w, "Stale artefacts (%d of %d):\n", staleCount, len(rep.StaleArtifacts))
	for _, a := range rep.StaleArtifacts {
		icon := "✓"
		if a.Stale {
			icon = "✗"
		}
		line := fmt.Sprintf("  %s %s", icon, a.File)
		if a.Reason != "" {
			line += " — " + a.Reason
		}
		fmt.Fprintln(w, line)
	}

	if len(rep.RerunTests) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Tests to re-run (%d):\n", len(rep.RerunTests))
		for _, t := range rep.RerunTests {
			fmt.Fprintf(w, "  %s\n", t)
		}
	}
	return nil
}
