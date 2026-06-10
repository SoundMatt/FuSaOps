// Package sbom aggregates the Software Bills of Materials produced by the
// x-FuSa toolchain into one cross-language SBOM.
//
// Each x-FuSa tool emits its own SBOM (e.g. go-FuSa from go.mod/go.sum). FuSaOps
// decodes each component's SBOM, merges the package lists across every language,
// de-duplicates on name+version, and renders both a native FuSaOps JSON view and
// an SPDX 2.3 document so the whole polyglot project has one bill of materials.
package sbom

import (
	"sort"
	"time"
)

// Package is a single dependency entry, normalised across tools.
//
//fusa:req REQ-FO-SBM001
type Package struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Hash     string `json:"hash,omitempty"`
	Language string `json:"language,omitempty"`
}

// Document mirrors the SBOM JSON a tool emits. Only the fields FuSaOps merges
// are decoded.
//
//fusa:req REQ-FO-SBM002
type Document struct {
	Module     string    `json:"module"`
	GoVersion  string    `json:"goVersion,omitempty"`
	Components []Package `json:"components"`
}

// ComponentSBOM is one language's contribution to the aggregate SBOM.
//
//fusa:req REQ-FO-SBM003
type ComponentSBOM struct {
	Language  string    `json:"language"`
	Tool      string    `json:"tool"`
	Available bool      `json:"available"`
	Skipped   string    `json:"skipped,omitempty"`
	Module    string    `json:"module,omitempty"`
	Packages  []Package `json:"packages,omitempty"`
}

// Aggregate is the merged cross-language SBOM.
//
//fusa:req REQ-FO-SBM004
type Aggregate struct {
	GeneratedAt   time.Time       `json:"generatedAt"`
	Root          string          `json:"root"`
	Project       string          `json:"project,omitempty"`
	Components    []ComponentSBOM `json:"components"`
	Packages      []Package       `json:"packages"`
	TotalPackages int             `json:"totalPackages"`
}

// New builds an Aggregate from component SBOMs, merging and de-duplicating their
// package lists on (name, version). Components are sorted by tool name and the
// merged package list by name then version for deterministic output. Each merged
// package keeps the language of the first component that contributed it.
//
//fusa:req REQ-FO-SBM005
func New(root, project string, components []ComponentSBOM) *Aggregate {
	sort.Slice(components, func(i, j int) bool { return components[i].Tool < components[j].Tool })

	type key struct{ name, version string }
	seen := make(map[key]struct{})
	var merged []Package
	for _, c := range components {
		for _, p := range c.Packages {
			k := key{p.name(), p.Version}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			if p.Language == "" {
				p.Language = c.Language
			}
			merged = append(merged, p)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Name != merged[j].Name {
			return merged[i].Name < merged[j].Name
		}
		return merged[i].Version < merged[j].Version
	})

	return &Aggregate{
		GeneratedAt:   time.Now().UTC(),
		Root:          root,
		Project:       project,
		Components:    components,
		Packages:      merged,
		TotalPackages: len(merged),
	}
}

// name returns the package name (helper so an empty name still produces a stable
// de-duplication key).
func (p Package) name() string { return p.Name }
