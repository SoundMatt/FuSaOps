// Package comp decodes and aggregates McCabe cyclomatic complexity (V(G))
// comp-reports produced by x-FuSa tools per §9.2 of the x-FuSa spec.
package comp

import "fmt"

// ReportFile is the canonical output filename per §9.2.
const ReportFile = "comp-report.json"

// Report is the decoded per-tool comp-report.json (§9.2 / §13 canonical schema).
//
//fusa:req REQ-FO-COMP001
type Report struct {
	Tool           string     `json:"tool,omitempty"`
	Language       string     `json:"language,omitempty"`
	ToolVersion    string     `json:"toolVersion,omitempty"`
	Threshold      int        `json:"threshold"`
	DAL            string     `json:"dal,omitempty"`
	TotalFunctions int        `json:"totalFunctions"`
	Violations     int        `json:"violations"`
	Results        []Function `json:"results,omitempty"`
}

// Function is a single function's V(G) measurement within a comp-report.
//
//fusa:req REQ-FO-COMP001
type Function struct {
	File             string `json:"file"`
	Line             int    `json:"line,omitempty"`
	Name             string `json:"name"`
	Complexity       int    `json:"complexity"`
	ExceedsThreshold bool   `json:"exceedsThreshold"`
}

// ComponentComp is one tool/language comp result inside the cross-language Aggregate.
//
//fusa:req REQ-FO-COMP002
type ComponentComp struct {
	Language  string  `json:"language"`
	Tool      string  `json:"tool"`
	Available bool    `json:"-"`
	Skipped   string  `json:"skipped,omitempty"`
	Report    *Report `json:"report,omitempty"`
}

// Aggregate is the cross-language cyclomatic complexity roll-up.
//
//fusa:req REQ-FO-COMP002
type Aggregate struct {
	Root           string          `json:"root,omitempty"`
	Project        string          `json:"project,omitempty"`
	Components     []ComponentComp `json:"components"`
	TotalFunctions int             `json:"totalFunctions"`
	Violations     int             `json:"violations"`
}

// New builds an Aggregate from per-component results, summing functions and
// violations across all components that successfully produced a report.
//
//fusa:req REQ-FO-COMP002
func New(root, project string, components []ComponentComp) *Aggregate {
	agg := &Aggregate{Root: root, Project: project, Components: components}
	for i := range components {
		if components[i].Report != nil {
			agg.TotalFunctions += components[i].Report.TotalFunctions
			agg.Violations += components[i].Report.Violations
		}
	}
	return agg
}

// HasViolations returns true when any component has at least one function that
// exceeds the configured complexity threshold.
//
//fusa:req REQ-FO-COMP002
func (a *Aggregate) HasViolations() bool { return a.Violations > 0 }

// DALThreshold returns the McCabe V(G) threshold for the given DAL string
// (DAL-A → 4, DAL-B → 10, DAL-C → 15, DAL-D → 20). These are a project/McCabe
// convention, not a normative DO-178C mandate — DO-178C does not prescribe
// specific cyclomatic-complexity limits.
// Returns 0 (no threshold) for an unrecognised or empty DAL.
//
//fusa:req REQ-FO-COMP001
func DALThreshold(dal string) int {
	switch dal {
	case "DAL-A":
		return 4
	case "DAL-B":
		return 10
	case "DAL-C":
		return 15
	case "DAL-D":
		return 20
	default:
		return 0
	}
}

// ValidateDAL returns an error for an unrecognised DAL level (empty is allowed).
//
//fusa:req REQ-FO-COMP001
func ValidateDAL(dal string) error {
	switch dal {
	case "", "DAL-A", "DAL-B", "DAL-C", "DAL-D":
		return nil
	default:
		return fmt.Errorf("comp: unrecognised DAL %q (want DAL-A|DAL-B|DAL-C|DAL-D)", dal)
	}
}
