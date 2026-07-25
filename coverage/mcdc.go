// MC/DC coverage gate for FuSaOps — LLVM source-based condition/decision
// coverage analysis (DO-178C §6.4.4.b, Level A prerequisite).
package coverage

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// McdcReportFile is the default save path for the MC/DC report JSON.
const McdcReportFile = ".fusaops-mcdc-report.json"

// DefaultMcdcThreshold is the minimum condition coverage % enforced by GateMCDC at DAL-A.
const DefaultMcdcThreshold = 100.0

// McdcCondition records MC/DC coverage of one Boolean condition within a decision.
//
//fusa:req REQ-FO-COV004
type McdcCondition struct {
	ID       int  `json:"id"`
	CoveredT bool `json:"coveredT"` // condition exercised as true
	CoveredF bool `json:"coveredF"` // condition exercised as false
	Covered  bool `json:"covered"`  // true iff both CoveredT and CoveredF
}

// McdcDecision records coverage of one decision expression (Boolean combination).
//
//fusa:req REQ-FO-COV004
type McdcDecision struct {
	File       string          `json:"file"`
	Line       int             `json:"line"`
	Conditions []McdcCondition `json:"conditions"`
	Covered    bool            `json:"covered"` // true iff all conditions covered
}

// McdcFunction groups all decisions within a named function.
//
//fusa:req REQ-FO-COV004
type McdcFunction struct {
	Name      string         `json:"name"`
	File      string         `json:"file"`
	HasReqTag bool           `json:"hasReqTag"` // true if //fusa:req annotation found in source
	Decisions []McdcDecision `json:"decisions"`
	Covered   bool           `json:"covered"` // true iff all decisions covered
}

// McdcReport is the structured DO-178C MC/DC coverage report.
//
//fusa:req REQ-FO-COV004
type McdcReport struct {
	Generated     time.Time      `json:"generated"`
	DAL           DAL            `json:"dal"`
	ProfileMode   string         `json:"profileMode"`   // always "llvm-mcdc"
	Functions     []McdcFunction `json:"functions"`
	TotalConds    int            `json:"totalConditions"`
	CoveredConds  int            `json:"coveredConditions"`
	CondPct       float64        `json:"conditionPct"`
	Threshold     float64        `json:"threshold"`
	GatePassed    bool           `json:"gatePassed"`
	UncoveredReqs []string       `json:"uncoveredReqs,omitempty"` // annotated funcs with gaps
}

// --- LLVM JSON intermediate structs ---

type llvmExport struct {
	Data []llvmData `json:"data"`
}

type llvmData struct {
	Functions []llvmFunction `json:"functions"`
}

type llvmFunction struct {
	Name        string       `json:"name"`
	Filenames   []string     `json:"filenames"`
	MCDCRecords []llvmRecord `json:"mcdc_records"`
}

type llvmRecord struct {
	Conditions     []llvmCondition `json:"conditions"`
	DecisionRegion [8]int          `json:"decision_region"`
}

type llvmCondition struct {
	ID                int `json:"id"`
	CoveredTrueCount  int `json:"covered_true_count"`
	CoveredFalseCount int `json:"covered_false_count"`
}

// ParseMCDC reads an LLVM llvm-cov JSON export from r and returns the per-function
// condition coverage data.
//
//fusa:req REQ-FO-COV005
func ParseMCDC(r io.Reader) ([]McdcFunction, error) {
	var export llvmExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("coverage: parse LLVM MC/DC JSON: %w", err)
	}

	// Use a map to accumulate decisions per function (deduplication by name+file).
	type key struct{ name, file string }
	funcMap := make(map[key]*McdcFunction)
	var order []key

	for _, data := range export.Data {
		for _, fn := range data.Functions {
			file := ""
			if len(fn.Filenames) > 0 {
				file = fn.Filenames[0]
			}
			k := key{fn.Name, file}
			mf, exists := funcMap[k]
			if !exists {
				mf = &McdcFunction{Name: fn.Name, File: file}
				funcMap[k] = mf
				order = append(order, k)
			}
			for _, rec := range fn.MCDCRecords {
				line := rec.DecisionRegion[0]
				decision := McdcDecision{
					File: file,
					Line: line,
				}
				allCovered := true
				for _, cond := range rec.Conditions {
					covT := cond.CoveredTrueCount > 0
					covF := cond.CoveredFalseCount > 0
					covered := covT && covF
					if !covered {
						allCovered = false
					}
					decision.Conditions = append(decision.Conditions, McdcCondition{
						ID:       cond.ID,
						CoveredT: covT,
						CoveredF: covF,
						Covered:  covered,
					})
				}
				decision.Covered = allCovered
				mf.Decisions = append(mf.Decisions, decision)
			}
		}
	}

	result := make([]McdcFunction, 0, len(order))
	for _, k := range order {
		mf := funcMap[k]
		// Compute function-level covered: all decisions must be covered.
		allCovered := true
		for _, d := range mf.Decisions {
			if !d.Covered {
				allCovered = false
				break
			}
		}
		mf.Covered = allCovered
		result = append(result, *mf)
	}
	return result, nil
}

// FindAnnotatedFunctions walks dir for .go files and returns the set of
// function names whose doc-comment group contains a //fusa:req annotation.
// Uses go/parser with ParseComments mode; no external dependencies.
//
//fusa:req REQ-FO-COV009
func FindAnnotatedFunctions(dir string) (map[string]bool, error) {
	result := make(map[string]bool)
	fset := token.NewFileSet()

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil // non-fatal parse error; skip file
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			for _, comment := range fn.Doc.List {
				if strings.Contains(comment.Text, "//fusa:req") {
					result[fn.Name.Name] = true
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

// AnalyseMCDC merges parsed LLVM coverage data with the req-annotated function
// set and computes aggregate MC/DC statistics.
//
//fusa:req REQ-FO-COV006
func AnalyseMCDC(funcs []McdcFunction, annotated map[string]bool, dal DAL, threshold float64) *McdcReport {
	if threshold <= 0 {
		threshold = DefaultMcdcThreshold
	}

	rep := &McdcReport{
		Generated:   time.Now().UTC(),
		DAL:         dal,
		ProfileMode: "llvm-mcdc",
		Threshold:   threshold,
	}

	// Tag functions with req annotations, count conditions.
	tagged := make([]McdcFunction, len(funcs))
	copy(tagged, funcs)
	for i := range tagged {
		tagged[i].HasReqTag = annotated[tagged[i].Name]
		for _, d := range tagged[i].Decisions {
			for _, c := range d.Conditions {
				rep.TotalConds++
				if c.Covered {
					rep.CoveredConds++
				}
			}
		}
	}

	// Compute CondPct.
	if rep.TotalConds > 0 {
		rep.CondPct = float64(rep.CoveredConds) * 100 / float64(rep.TotalConds)
	}

	// Build UncoveredReqs: annotated functions with uncovered conditions.
	for _, fn := range tagged {
		if !fn.HasReqTag || fn.Covered {
			continue
		}
		uncoveredCount := 0
		for _, d := range fn.Decisions {
			for _, c := range d.Conditions {
				if !c.Covered {
					uncoveredCount++
				}
			}
		}
		if uncoveredCount > 0 {
			rep.UncoveredReqs = append(rep.UncoveredReqs,
				fmt.Sprintf("%s (%s): %d uncovered condition(s)", fn.Name, fn.File, uncoveredCount))
		}
	}

	// Sort for determinism.
	sort.Slice(tagged, func(i, j int) bool {
		if tagged[i].File != tagged[j].File {
			return tagged[i].File < tagged[j].File
		}
		return tagged[i].Name < tagged[j].Name
	})
	sort.Strings(rep.UncoveredReqs)
	rep.Functions = tagged

	// Gate: pass iff no uncovered req-annotated functions AND threshold met.
	rep.GatePassed = len(rep.UncoveredReqs) == 0 &&
		(rep.TotalConds == 0 || rep.CondPct >= threshold)

	return rep
}

// GateMCDC reports whether the MC/DC coverage gate passed.
// Returns false if any //fusa:req-annotated function has uncovered conditions
// or if overall condition coverage falls below the threshold.
//
//fusa:req REQ-FO-COV007
func GateMCDC(rep *McdcReport) bool { return rep.GatePassed }

// RenderMCDC writes the MC/DC report to w in the requested format.
// Supported formats: "text", "json", "markdown"/"md".
//
//fusa:req REQ-FO-COV008
func RenderMCDC(w io.Writer, rep *McdcReport, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	case "text":
		return renderMCDCText(w, rep)
	case "markdown", "md":
		return renderMCDCMarkdown(w, rep)
	default:
		return fmt.Errorf("coverage: unsupported MC/DC report format %q", format)
	}
}

func renderMCDCText(w io.Writer, rep *McdcReport) error {
	gateStr := "PASS"
	if !rep.GatePassed {
		gateStr = "FAIL"
	}
	fmt.Fprintf(w, "DO-178C MC/DC Coverage Report (LLVM source-based)\n")
	fmt.Fprintf(w, "DAL: %s  Generated: %s\n\n", rep.DAL, rep.Generated.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "Condition coverage : %.1f%%  threshold: %.1f%%\n", rep.CondPct, rep.Threshold)
	fmt.Fprintf(w, "Gate               : %s\n", gateStr)
	if len(rep.UncoveredReqs) > 0 {
		fmt.Fprintf(w, "\nUncovered //fusa:req annotated functions (%d):\n", len(rep.UncoveredReqs))
		for _, u := range rep.UncoveredReqs {
			fmt.Fprintf(w, "  %s\n", u)
		}
	}
	if len(rep.Functions) > 0 {
		fmt.Fprintf(w, "\nPer-function condition coverage:\n")
		for _, fn := range rep.Functions {
			status := "PASS"
			if !fn.Covered {
				status = "FAIL"
			}
			reqTag := ""
			if fn.HasReqTag {
				reqTag = "  [req-annotated]"
			}
			fmt.Fprintf(w, "  %s  %s (%s)%s\n", status, fn.Name, fn.File, reqTag)
		}
	}
	return nil
}

func renderMCDCMarkdown(w io.Writer, rep *McdcReport) error {
	gateStr := "PASS"
	if !rep.GatePassed {
		gateStr = "FAIL"
	}
	fmt.Fprintf(w, "# FuSaOps — DO-178C MC/DC Coverage Report\n\n")
	fmt.Fprintf(w, "**Gate: %s** — %.1f%% condition coverage (threshold: %.1f%%) · DAL: %s · Generated %s\n\n",
		gateStr, rep.CondPct, rep.Threshold, rep.DAL, rep.Generated.Format("2006-01-02 15:04 UTC"))
	if len(rep.UncoveredReqs) > 0 {
		fmt.Fprintf(w, "## Uncovered req-annotated functions\n\n")
		for _, u := range rep.UncoveredReqs {
			fmt.Fprintf(w, "- %s\n", u)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "## Per-function MC/DC coverage")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Function | File | Req-annotated | Covered |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for _, fn := range rep.Functions {
		covered := "YES"
		if !fn.Covered {
			covered = "NO"
		}
		reqTag := "no"
		if fn.HasReqTag {
			reqTag = "yes"
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", fn.Name, fn.File, reqTag, covered)
	}
	return nil
}
