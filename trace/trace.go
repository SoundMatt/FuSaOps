// Package trace aggregates the per-language requirement traceability matrices
// produced by the x-FuSa toolchain into a single cross-language view.
//
// Each x-FuSa tool emits a requirement traceability matrix from
// "<tool> trace --format json" and a qualification summary from
// "<tool> qualify". FuSaOps does not compute traceability itself: it decodes
// each tool's matrix, rolls the coverage figures up across every language, and
// records each component's qualification status so a polyglot project gets one
// auditor-ready coverage and tool-confidence view.
package trace

import (
	"sort"
	"time"
)

// Requirement mirrors one entry of a tool's requirement registry. Only the
// fields FuSaOps surfaces in the cross-language view are decoded.
//
//fusa:req REQ-FO-TRC001
type Requirement struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Standard string `json:"standard,omitempty"`
	Level    string `json:"level,omitempty"`
	ASIL     string `json:"asil,omitempty"`
}

// Tag is a single //fusa:req or //fusa:test annotation discovered in a
// component's source. Kind distinguishes a requirement reference from a test.
//
//fusa:req REQ-FO-TRC002
type Tag struct {
	RequirementID string `json:"requirementId"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	Kind          string `json:"kind"`
}

// Coverage holds the headline traceability figures from a tool's matrix.
//
//fusa:req REQ-FO-TRC003
type Coverage struct {
	TotalRequirements  int `json:"totalRequirements"`
	TracedRequirements int `json:"tracedRequirements"`
	TestedRequirements int `json:"testedRequirements"`
}

// Matrix mirrors the JSON document a tool emits from "trace --format json".
//
//fusa:req REQ-FO-TRC004
type Matrix struct {
	Requirements []Requirement `json:"requirements"`
	Tags         []Tag         `json:"tags"`
	Coverage     Coverage      `json:"coverage"`
}

// Qualification mirrors the headline figures of a tool's qualification report
// ("<tool> qualify"). It is the tool-confidence evidence (ISO 26262 §11) rolled
// up alongside requirement coverage.
//
//fusa:req REQ-FO-TRC005
type Qualification struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// ComponentTrace is one language's contribution to the aggregate matrix.
//
//fusa:req REQ-FO-TRC006
type ComponentTrace struct {
	Language      string         `json:"language"`
	Tool          string         `json:"tool"`
	Available     bool           `json:"available"`
	Skipped       string         `json:"skipped,omitempty"`
	Coverage      Coverage       `json:"coverage"`
	Requirements  []Requirement  `json:"requirements,omitempty"`
	Qualification *Qualification `json:"qualification,omitempty"`
}

// pct returns whole-number percent of n over total, guarding division by zero.
func pct(n, total int) int {
	if total == 0 {
		return 100
	}
	return n * 100 / total
}

// TracedPct is the percentage of this component's requirements that are traced.
//
//fusa:req REQ-FO-TRC007
func (c ComponentTrace) TracedPct() int {
	return pct(c.Coverage.TracedRequirements, c.Coverage.TotalRequirements)
}

// TestedPct is the percentage of this component's requirements that are tested.
//
//fusa:req REQ-FO-TRC007
func (c ComponentTrace) TestedPct() int {
	return pct(c.Coverage.TestedRequirements, c.Coverage.TotalRequirements)
}

// AggregateCoverage sums coverage across every component.
//
//fusa:req REQ-FO-TRC008
type AggregateCoverage struct {
	TotalRequirements  int `json:"totalRequirements"`
	TracedRequirements int `json:"tracedRequirements"`
	TestedRequirements int `json:"testedRequirements"`
	TracedPct          int `json:"tracedPct"`
	TestedPct          int `json:"testedPct"`
}

// Aggregate is the cross-language traceability roll-up.
//
//fusa:req REQ-FO-TRC009
type Aggregate struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	Root        string            `json:"root"`
	Project     string            `json:"project,omitempty"`
	Components  []ComponentTrace  `json:"components"`
	Coverage    AggregateCoverage `json:"coverage"`
}

// New builds an Aggregate from component traces, summing coverage across every
// component whose matrix was collected. Components are sorted by tool name for
// deterministic output. Skipped components contribute nothing to the totals, so
// coverage gaps caused by a missing tool stay visible rather than inflating the
// percentage.
//
//fusa:req REQ-FO-TRC010
func New(root, project string, components []ComponentTrace) *Aggregate {
	sort.Slice(components, func(i, j int) bool { return components[i].Tool < components[j].Tool })
	a := &Aggregate{
		GeneratedAt: time.Now().UTC(),
		Root:        root,
		Project:     project,
		Components:  components,
	}
	for _, c := range components {
		if c.Skipped != "" {
			continue // a missing tool must not inflate or shrink coverage
		}
		a.Coverage.TotalRequirements += c.Coverage.TotalRequirements
		a.Coverage.TracedRequirements += c.Coverage.TracedRequirements
		a.Coverage.TestedRequirements += c.Coverage.TestedRequirements
	}
	a.Coverage.TracedPct = pct(a.Coverage.TracedRequirements, a.Coverage.TotalRequirements)
	a.Coverage.TestedPct = pct(a.Coverage.TestedRequirements, a.Coverage.TotalRequirements)
	return a
}

// HasGaps reports whether any requirement across all languages is untraced or
// untested. It is the CI gate for "fusaops trace".
//
//fusa:req REQ-FO-TRC011
func (a *Aggregate) HasGaps() bool {
	c := a.Coverage
	return c.TracedRequirements < c.TotalRequirements ||
		c.TestedRequirements < c.TotalRequirements
}

// Status returns the overall PASS/GAP verdict for the aggregate.
//
//fusa:req REQ-FO-TRC011
func (a *Aggregate) Status() string {
	if a.HasGaps() {
		return "GAP"
	}
	return "PASS"
}
