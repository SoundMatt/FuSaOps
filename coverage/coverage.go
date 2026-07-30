// Package coverage reads Go coverage profiles and produces DO-178C-style
// structural coverage reports (DO-178C §6.4.4, Annex A Table A-7).
//
// It reports statement coverage (always available) and estimates decision
// coverage from block data where present, and flags whether MC/DC evidence
// is required (DAL-A) but cannot be automatically verified.
package coverage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CoverageFile is the default Go coverage profile filename.
const CoverageFile = "coverage.out"

// DAL represents a DO-178C Design Assurance Level.
type DAL string

const (
	DALA DAL = "DAL-A"
	DALB DAL = "DAL-B"
	DALC DAL = "DAL-C"
	DALD DAL = "DAL-D"
)

// Block is a single coverage block from the Go profile.
type Block struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Stmts     int    `json:"stmts"`
	Count     int    `json:"count"`
}

// FileStats holds per-file coverage statistics.
type FileStats struct {
	File    string  `json:"file"`
	Stmts   int     `json:"stmts"`
	Covered int     `json:"covered"`
	StmtPct float64 `json:"stmtPct"`
}

// Report is the DO-178C structural coverage report.
//
//fusa:req REQ-FO-COV001
type Report struct {
	Generated        time.Time   `json:"generated"`
	DAL              DAL         `json:"dal"`
	StmtTotal        int         `json:"stmtTotal"`
	StmtCovered      int         `json:"stmtCovered"`
	StmtPct          float64     `json:"stmtPct"`
	StmtRequired     bool        `json:"stmtRequired"`
	DecisionPct      float64     `json:"decisionPct,omitempty"`
	DecisionNote     string      `json:"decisionNote,omitempty"`
	DecisionRequired bool        `json:"decisionRequired"`
	MCDCRequired     bool        `json:"mcdcRequired"`
	MCDCNote         string      `json:"mcdcNote,omitempty"`
	Files            []FileStats `json:"files"`
	Gaps             []string    `json:"gaps,omitempty"`
}

// Parse reads a Go coverage profile from r and returns the raw blocks.
//
//fusa:req REQ-FO-COV002
func Parse(r io.Reader) ([]Block, error) {
	scanner := bufio.NewScanner(r)
	var blocks []Block
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "mode:") || line == "" {
			continue
		}
		// format: pkg/file.go:startLine.col,endLine.col numStmts count
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		file := line[:colon]
		rest := line[colon+1:]
		parts := strings.Fields(rest)
		if len(parts) != 3 {
			continue
		}
		rangePart := parts[0]
		dash := strings.Index(rangePart, ",")
		if dash < 0 {
			continue
		}
		startLine, _ := strconv.Atoi(strings.Split(rangePart[:dash], ".")[0])
		endLine, _ := strconv.Atoi(strings.Split(rangePart[dash+1:], ".")[0])
		stmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])
		blocks = append(blocks, Block{
			File:      file,
			StartLine: startLine,
			EndLine:   endLine,
			Stmts:     stmts,
			Count:     count,
		})
	}
	return blocks, scanner.Err()
}

// Analyse computes a DO-178C coverage report from blocks for the given DAL.
//
//fusa:req REQ-FO-COV001
func Analyse(blocks []Block, dal DAL) *Report {
	rep := &Report{
		Generated:        time.Now().UTC(),
		DAL:              dal,
		StmtRequired:     true,
		DecisionRequired: dal == DALA || dal == DALB,
		MCDCRequired:     dal == DALA,
	}

	fileMap := make(map[string]*FileStats)
	for _, b := range blocks {
		fs, ok := fileMap[b.File]
		if !ok {
			fs = &FileStats{File: b.File}
			fileMap[b.File] = fs
		}
		fs.Stmts += b.Stmts
		rep.StmtTotal += b.Stmts
		if b.Count > 0 {
			fs.Covered += b.Stmts
			rep.StmtCovered += b.Stmts
		}
	}

	if rep.StmtTotal > 0 {
		rep.StmtPct = float64(rep.StmtCovered) * 100 / float64(rep.StmtTotal)
	}

	for _, fs := range fileMap {
		if fs.Stmts > 0 {
			fs.StmtPct = float64(fs.Covered) * 100 / float64(fs.Stmts)
		}
		rep.Files = append(rep.Files, *fs)
		if fs.StmtPct < 100 {
			rep.Gaps = append(rep.Gaps, fmt.Sprintf("%s: %.1f%% statement coverage", fs.File, fs.StmtPct))
		}
	}
	sort.Slice(rep.Files, func(i, j int) bool { return rep.Files[i].File < rep.Files[j].File })
	sort.Strings(rep.Gaps)

	// Decision coverage: this is NOT true decision (branch true/false) coverage.
	// It is the block-hit ratio (an upper-bound proxy), surfaced here only as an
	// indicator. Real DC/MC-DC evidence requires the --mcdc path below.
	totalBlocks, coveredBlocks := 0, 0
	for _, b := range blocks {
		totalBlocks++
		if b.Count > 0 {
			coveredBlocks++
		}
	}
	if totalBlocks > 0 {
		rep.DecisionPct = float64(coveredBlocks) * 100 / float64(totalBlocks)
		rep.DecisionNote = "block-coverage proxy — NOT true decision coverage; provide MC/DC data for certification-grade DC evidence"
	}

	if rep.MCDCRequired {
		rep.MCDCNote = "Run 'fusaops coverage --mcdc --mcdc-file <llvm-coverage.json>' to enable automated MC/DC gate (DAL-A prerequisite)"
	}

	return rep
}

// BuildFromFile reads a Go coverage profile file and returns an analysis report.
//
//fusa:req REQ-FO-COV002
func BuildFromFile(path string, dal DAL) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("coverage: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	blocks, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("coverage: parse: %w", err)
	}
	return Analyse(blocks, dal), nil
}

// Render writes the coverage report in the requested format to w.
// Supported formats: "text", "json", "markdown" (alias "md").
//
//fusa:req REQ-FO-COV003
func Render(w io.Writer, rep *Report, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	case "text":
		return renderText(w, rep)
	case "markdown", "md":
		return renderMarkdown(w, rep)
	default:
		return fmt.Errorf("coverage: unsupported format %q", format)
	}
}

func renderText(w io.Writer, rep *Report) error {
	fmt.Fprintf(w, "DO-178C Structural Coverage Report\n")
	fmt.Fprintf(w, "DAL: %s  Generated: %s\n\n", rep.DAL, rep.Generated.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "Statement coverage : %5.1f%%  (required: %s)\n", rep.StmtPct, yesNo(rep.StmtRequired))
	fmt.Fprintf(w, "Decision coverage  : %5.1f%%  (required: %s)\n", rep.DecisionPct, yesNo(rep.DecisionRequired))
	if rep.DecisionNote != "" {
		fmt.Fprintf(w, "  Note: %s\n", rep.DecisionNote)
	}
	if rep.MCDCRequired {
		fmt.Fprintf(w, "MC/DC coverage     : MANUAL CHECK REQUIRED\n")
		fmt.Fprintf(w, "  Note: %s\n", rep.MCDCNote)
	}
	if len(rep.Gaps) > 0 {
		fmt.Fprintf(w, "\nCoverage gaps (%d file(s)):\n", len(rep.Gaps))
		for _, g := range rep.Gaps {
			fmt.Fprintf(w, "  %s\n", g)
		}
	}
	fmt.Fprintf(w, "\nPer-file statement coverage:\n")
	for _, fs := range rep.Files {
		fmt.Fprintf(w, "  %5.1f%%  %s\n", fs.StmtPct, fs.File)
	}
	return nil
}

func renderMarkdown(w io.Writer, rep *Report) error {
	badge := "🟢"
	if rep.StmtPct < 80 {
		badge = "🔴"
	} else if rep.StmtPct < 100 {
		badge = "🟡"
	}
	fmt.Fprintf(w, "# FuSaOps — DO-178C Structural Coverage\n\n")
	fmt.Fprintf(w, "%s **%.1f%%** statement coverage · DAL: %s · Generated %s\n\n",
		badge, rep.StmtPct, rep.DAL, rep.Generated.Format("2006-01-02 15:04 UTC"))
	fmt.Fprintln(w, "| Metric | Value | Required |")
	fmt.Fprintln(w, "|---|---:|---|")
	fmt.Fprintf(w, "| Statement coverage | %.1f%% | %s |\n", rep.StmtPct, yesNo(rep.StmtRequired))
	fmt.Fprintf(w, "| Decision coverage | %.1f%% | %s |\n", rep.DecisionPct, yesNo(rep.DecisionRequired))
	mcdcVal := "n/a"
	if rep.MCDCRequired {
		mcdcVal = "MANUAL CHECK REQUIRED"
	}
	fmt.Fprintf(w, "| MC/DC coverage | %s | %s |\n", mcdcVal, yesNo(rep.MCDCRequired))
	if rep.DecisionNote != "" {
		fmt.Fprintf(w, "\n_Note: %s_\n", rep.DecisionNote)
	}
	if len(rep.Gaps) > 0 {
		fmt.Fprintf(w, "\n## Coverage gaps (%d file(s))\n\n", len(rep.Gaps))
		fmt.Fprintln(w, "| File | Statement % |")
		fmt.Fprintln(w, "|---|---:|")
		for _, fs := range rep.Files {
			if fs.StmtPct < 100 {
				fmt.Fprintf(w, "| %s | %.1f%% |\n", fs.File, fs.StmtPct)
			}
		}
	}
	return nil
}

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "no"
}
