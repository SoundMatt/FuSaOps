// Package mcdc decodes and aggregates Modified Condition/Decision Coverage (MC/DC)
// reports produced by x-FuSa tools per §9.4 of the x-FuSa spec.
//
// MC/DC is required for DO-178C Level A software and IS0 26262 ASIL D. Each
// x-FuSa tool that supports the --mcdc flag emits a structured MC/DC report;
// FuSaOps collects those per-tool reports and rolls them up into a single
// cross-language MCDCAggregate.
package mcdc

// ReportFile is the canonical output filename per §9.4.
const ReportFile = "mcdc-report.json"

// Condition is a single boolean sub-expression tracked in an MC/DC report.
//
//fusa:req REQ-FO-MCDC001
type Condition struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Covered bool   `json:"covered"`
}

// Decision is a logical decision point (branch) and its constituent conditions.
//
//fusa:req REQ-FO-MCDC001
type Decision struct {
	Name       string      `json:"name"`
	File       string      `json:"file"`
	Line       int         `json:"line,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
	// MCDCCovered is true when full MC/DC coverage has been demonstrated for
	// every condition in the decision.
	MCDCCovered bool `json:"mcdcCovered"`
}

// Report is the decoded per-tool MC/DC report emitted by an x-FuSa tool.
//
//fusa:req REQ-FO-MCDC001
type Report struct {
	Tool              string     `json:"tool,omitempty"`
	Language          string     `json:"language,omitempty"`
	ToolVersion       string     `json:"toolVersion,omitempty"`
	TotalConditions   int        `json:"totalConditions"`
	CoveredConditions int        `json:"coveredConditions"`
	TotalDecisions    int        `json:"totalDecisions"`
	CoveredDecisions  int        `json:"coveredDecisions"`
	GatePassed        bool       `json:"gatePassed"`
	Decisions         []Decision `json:"decisions,omitempty"`
}

// CoveragePct returns the whole-number MC/DC condition-coverage percentage.
//
//fusa:req REQ-FO-MCDC004
func (r *Report) CoveragePct() int {
	if r.TotalConditions == 0 {
		return 100
	}
	return r.CoveredConditions * 100 / r.TotalConditions
}

// MCDCComponent is one tool/language MC/DC result inside the cross-language Aggregate.
//
//fusa:req REQ-FO-MCDC002
type MCDCComponent struct {
	Language  string  `json:"language"`
	Tool      string  `json:"tool"`
	Available bool    `json:"-"`
	Skipped   string  `json:"skipped,omitempty"`
	Report    *Report `json:"report,omitempty"`
}

// MCDCAggregate is the cross-language MC/DC roll-up.
//
//fusa:req REQ-FO-MCDC002
type MCDCAggregate struct {
	Root              string          `json:"root,omitempty"`
	Project           string          `json:"project,omitempty"`
	Components        []MCDCComponent `json:"components"`
	TotalConditions   int             `json:"totalConditions"`
	CoveredConditions int             `json:"coveredConditions"`
	GatePassed        bool            `json:"gatePassed"`
}

// New builds an MCDCAggregate from per-component results, summing condition
// counts across all components that successfully produced a report. GatePassed
// is true only when every non-skipped component's gate passed.
//
//fusa:req REQ-FO-MCDC002
func New(root, project string, components []MCDCComponent) *MCDCAggregate {
	agg := &MCDCAggregate{
		Root:       root,
		Project:    project,
		Components: components,
		GatePassed: true, // start optimistic; any failing component flips this
	}
	hasData := false
	for i := range components {
		if components[i].Report != nil {
			agg.TotalConditions += components[i].Report.TotalConditions
			agg.CoveredConditions += components[i].Report.CoveredConditions
			if !components[i].Report.GatePassed {
				agg.GatePassed = false
			}
			hasData = true
		}
	}
	if !hasData {
		agg.GatePassed = false
	}
	return agg
}

// CoveragePct returns the whole-number cross-language MC/DC coverage percentage.
//
//fusa:req REQ-FO-MCDC004
func (a *MCDCAggregate) CoveragePct() int {
	if a.TotalConditions == 0 {
		return 100
	}
	return a.CoveredConditions * 100 / a.TotalConditions
}
