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
//fusa:req REQ-FO-TRC019
type Requirement struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Standard string `json:"standard,omitempty"`
	Level    string `json:"level,omitempty"`
	ASIL     string `json:"asil,omitempty"`
	Status   string `json:"status,omitempty"` // covered | untraced | untested (§5)
	Parent   string `json:"parent,omitempty"` // ID of parent HLR; set for LLR requirements
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
	TotalRequirements     int `json:"totalRequirements"`
	TracedRequirements    int `json:"tracedRequirements"`
	TestedRequirements    int `json:"testedRequirements"`
	SecTestedRequirements int `json:"secTestedRequirements,omitempty"` // §5
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
	Tags          []Tag          `json:"tags,omitempty"`
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

// SecTestedPct is the percentage of requirements covered by security tests.
//
//fusa:req REQ-FO-TRC016
func (c ComponentTrace) SecTestedPct() int {
	return pct(c.Coverage.SecTestedRequirements, c.Coverage.TotalRequirements)
}

// AggregateCoverage sums coverage across every component.
//
//fusa:req REQ-FO-TRC008
type AggregateCoverage struct {
	TotalRequirements     int `json:"totalRequirements"`
	TracedRequirements    int `json:"tracedRequirements"`
	TestedRequirements    int `json:"testedRequirements"`
	SecTestedRequirements int `json:"secTestedRequirements,omitempty"`
	TracedPct             int `json:"tracedPct"`
	TestedPct             int `json:"testedPct"`
	SecTestedPct          int `json:"secTestedPct,omitempty"`
}

// Aggregate is the cross-language traceability roll-up.
//
//fusa:req REQ-FO-TRC009
//fusa:req REQ-FO-TRC020
type Aggregate struct {
	GeneratedAt   time.Time             `json:"generatedAt"`
	Root          string                `json:"root"`
	Project       string                `json:"project,omitempty"`
	Components    []ComponentTrace      `json:"components"`
	Coverage      AggregateCoverage     `json:"coverage"`
	Decomposition *DecompositionReport  `json:"decomposition,omitempty"`
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
		a.Coverage.SecTestedRequirements += c.Coverage.SecTestedRequirements
	}
	a.Coverage.TracedPct = pct(a.Coverage.TracedRequirements, a.Coverage.TotalRequirements)
	a.Coverage.TestedPct = pct(a.Coverage.TestedRequirements, a.Coverage.TotalRequirements)
	a.Coverage.SecTestedPct = pct(a.Coverage.SecTestedRequirements, a.Coverage.TotalRequirements)
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

// FilterGaps returns a copy of a with each component's Requirements slice
// reduced to only those that are untraced or untested. Coverage totals are
// preserved so the aggregate gate is unchanged. Used by "fusaops trace --gaps".
//
//fusa:req REQ-FO-TRC017
func FilterGaps(a *Aggregate) *Aggregate {
	out := *a
	out.Components = make([]ComponentTrace, len(a.Components))
	for i, c := range a.Components {
		traced := map[string]bool{}
		tested := map[string]bool{}
		for _, t := range c.Tags {
			switch t.Kind {
			case "impl", "req":
				traced[t.RequirementID] = true
			case "test", "sec-test":
				tested[t.RequirementID] = true
			}
		}
		fc := c
		fc.Requirements = nil
		for _, r := range c.Requirements {
			if r.Status == "covered" {
				continue
			}
			if r.Status == "" && traced[r.ID] && tested[r.ID] {
				continue
			}
			fc.Requirements = append(fc.Requirements, r)
		}
		out.Components[i] = fc
	}
	return &out
}
