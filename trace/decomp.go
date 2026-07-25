package trace

import (
	"fmt"
	"sort"
	"strings"
)

// HLR/LLR level constants used in Requirement.Level.
//
//fusa:req REQ-FO-TRC019
const (
	LevelHLR = "HLR"
	LevelLLR = "LLR"
)

// DecompositionViolation describes a single HLR/LLR decomposition defect.
//
//fusa:req REQ-FO-TRC020
type DecompositionViolation struct {
	// Kind is one of "orphan-llr", "unparented-llr", or "childless-hlr".
	Kind          string `json:"kind"`
	RequirementID string `json:"requirementId"`
	ParentID      string `json:"parentId,omitempty"` // set for "orphan-llr"
	Component     string `json:"component"`          // tool name
}

// String returns a human-readable description of the violation.
//
//fusa:req REQ-FO-TRC020
func (v DecompositionViolation) String() string {
	switch v.Kind {
	case "orphan-llr":
		return fmt.Sprintf("[%s] LLR %s references unknown HLR %s", v.Component, v.RequirementID, v.ParentID)
	case "unparented-llr":
		return fmt.Sprintf("[%s] LLR %s has no parent HLR", v.Component, v.RequirementID)
	case "childless-hlr":
		return fmt.Sprintf("[%s] HLR %s has no LLR children", v.Component, v.RequirementID)
	default:
		return fmt.Sprintf("[%s] %s: %s", v.Component, v.Kind, v.RequirementID)
	}
}

// DecompositionReport summarises the HLR/LLR decomposition gate results.
//
//fusa:req REQ-FO-TRC020
type DecompositionReport struct {
	HLRCount   int                      `json:"hlrCount"`
	LLRCount   int                      `json:"llrCount"`
	Violations []DecompositionViolation `json:"violations"`
}

// Valid reports whether the decomposition gate passed (no violations).
//
//fusa:req REQ-FO-TRC020
func (r *DecompositionReport) Valid() bool {
	return r == nil || len(r.Violations) == 0
}

// reqMeta holds the fields needed to run the decomposition gate for one
// requirement (gathered from all non-skipped components).
type reqMeta struct {
	Level     string
	Parent    string
	Component string
}

// CheckDecomposition inspects all non-skipped component requirements and
// verifies that every LLR references a known HLR parent and every HLR has at
// least one LLR child. Requirements whose Level is empty are ignored, keeping
// legacy matrices unaffected.
//
//fusa:req REQ-FO-TRC021
func CheckDecomposition(a *Aggregate) *DecompositionReport {
	// Build a global map of id → reqMeta from non-skipped components.
	global := make(map[string]reqMeta)
	for _, c := range a.Components {
		if c.Skipped != "" {
			continue
		}
		for _, r := range c.Requirements {
			if r.Level != LevelHLR && r.Level != LevelLLR {
				continue
			}
			global[r.ID] = reqMeta{
				Level:     r.Level,
				Parent:    r.Parent,
				Component: c.Tool,
			}
		}
	}

	// Build a set of all known HLR IDs (cross-component).
	hlrIDs := make(map[string]bool)
	for id, m := range global {
		if m.Level == LevelHLR {
			hlrIDs[id] = true
		}
	}

	// Track how many LLR children each HLR has.
	hlrChildCount := make(map[string]int)

	var violations []DecompositionViolation

	// Check LLR requirements.
	for id, m := range global {
		if m.Level != LevelLLR {
			continue
		}
		switch {
		case m.Parent == "":
			violations = append(violations, DecompositionViolation{
				Kind:          "unparented-llr",
				RequirementID: id,
				Component:     m.Component,
			})
		case !hlrIDs[m.Parent]:
			violations = append(violations, DecompositionViolation{
				Kind:          "orphan-llr",
				RequirementID: id,
				ParentID:      m.Parent,
				Component:     m.Component,
			})
		default:
			hlrChildCount[m.Parent]++
		}
	}

	// Check HLR requirements.
	for id, m := range global {
		if m.Level != LevelHLR {
			continue
		}
		if hlrChildCount[id] == 0 {
			violations = append(violations, DecompositionViolation{
				Kind:          "childless-hlr",
				RequirementID: id,
				Component:     m.Component,
			})
		}
	}

	// Sort violations deterministically by (Component, RequirementID, Kind).
	sort.Slice(violations, func(i, j int) bool {
		a, b := violations[i], violations[j]
		if a.Component != b.Component {
			return a.Component < b.Component
		}
		if a.RequirementID != b.RequirementID {
			return a.RequirementID < b.RequirementID
		}
		return a.Kind < b.Kind
	})

	hlrCount := 0
	llrCount := 0
	for _, m := range global {
		switch m.Level {
		case LevelHLR:
			hlrCount++
		case LevelLLR:
			llrCount++
		}
	}

	return &DecompositionReport{
		HLRCount:   hlrCount,
		LLRCount:   llrCount,
		Violations: violations,
	}
}

// SeverityForDecomposition maps the enforce flag and project integrity level
// to one of "error", "warn", or "off".
//
// enforce values "error", "warn", and "off" are passed through directly.
// "auto" or "" derive the severity from the project's integrity level:
//   - DAL-A or DAL-B → "error"  (DO-178C highest criticality)
//   - DAL-C, DAL-D, DAL-E → "warn"
//   - ASIL-D or ASIL-C → "error"  (ISO 26262 highest criticality)
//   - ASIL-B or ASIL-A → "warn"
//   - No integrity level configured → "warn" (safe default)
//
//fusa:req REQ-FO-TRC022
func SeverityForDecomposition(enforce, dal, asil string) string {
	switch enforce {
	case "error", "warn", "off":
		return enforce
	}
	// "auto" or "" → derive from integrity level.
	switch strings.ToUpper(dal) {
	case "DAL-A", "DAL-B":
		return "error"
	case "DAL-C", "DAL-D", "DAL-E":
		return "warn"
	}
	switch strings.ToUpper(asil) {
	case "ASIL-D", "ASIL-C":
		return "error"
	case "ASIL-B", "ASIL-A":
		return "warn"
	}
	return "warn"
}
