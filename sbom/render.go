package sbom

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Render writes the aggregate SBOM to w in the requested format.
//
//fusa:req REQ-FO-SBM006
func Render(w io.Writer, a *Aggregate, format string) error {
	switch format {
	case "", "json":
		return renderJSON(w, a)
	case "text":
		return renderText(w, a)
	case "spdx":
		return renderSPDX(w, a)
	default:
		return fmt.Errorf("sbom: unsupported format %q", format)
	}
}

// RenderToFile writes the SBOM to path in format, or to stdout if path empty.
//
//fusa:req REQ-FO-SBM006
func RenderToFile(a *Aggregate, format, path string) error {
	if path == "" {
		return Render(os.Stdout, a, format)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("sbom: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Render(f, a, format)
}

//fusa:req REQ-FO-SBM007
func renderJSON(w io.Writer, a *Aggregate) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("sbom: json encode: %w", err)
	}
	return nil
}

//fusa:req REQ-FO-SBM008
func renderText(w io.Writer, a *Aggregate) error {
	fmt.Fprintln(w, "FuSaOps Aggregate SBOM")
	fmt.Fprintf(w, "Generated: %s\n", a.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	if a.Project != "" {
		fmt.Fprintf(w, "Project:   %s\n", a.Project)
	}
	fmt.Fprintf(w, "Packages:  %d across %d component(s)\n\n", a.TotalPackages, len(a.Components))
	for _, c := range a.Components {
		if c.Skipped != "" {
			fmt.Fprintf(w, "── %s (%s) ── skipped: %s\n", c.Tool, c.Language, c.Skipped)
			continue
		}
		fmt.Fprintf(w, "── %s (%s) ── %s · %d package(s)\n", c.Tool, c.Language, c.Module, len(c.Packages))
	}
	fmt.Fprintln(w)
	for _, p := range a.Packages {
		fmt.Fprintf(w, "  %-40s %-16s [%s]\n", p.Name, p.Version, p.Language)
	}
	return nil
}

// spdxDocument is a minimal, valid SPDX 2.3 JSON document.
type spdxDocument struct {
	SPDXVersion       string             `json:"spdxVersion"`
	DataLicense       string             `json:"dataLicense"`
	SPDXID            string             `json:"SPDXID"`
	Name              string             `json:"name"`
	DocumentNamespace string             `json:"documentNamespace"`
	CreationInfo      spdxCreationInfo   `json:"creationInfo"`
	Packages          []spdxPackage      `json:"packages"`
	Relationships     []spdxRelationship `json:"relationships"`
}

type spdxCreationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	Name             string `json:"name"`
	SPDXID           string `json:"SPDXID"`
	VersionInfo      string `json:"versionInfo,omitempty"`
	DownloadLocation string `json:"downloadLocation"`
	FilesAnalyzed    bool   `json:"filesAnalyzed"`
}

type spdxRelationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
	RelationshipType   string `json:"relationshipType"`
}

//fusa:req REQ-FO-SBM009
func renderSPDX(w io.Writer, a *Aggregate) error {
	name := a.Project
	if name == "" {
		name = "fusaops-project"
	}
	created := a.GeneratedAt.UTC().Format(time.RFC3339)
	doc := spdxDocument{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              name,
		DocumentNamespace: fmt.Sprintf("https://soundmatt.github.io/fusaops/spdx/%s-%d", name, a.GeneratedAt.UnixNano()),
		CreationInfo: spdxCreationInfo{
			Created:  created,
			Creators: []string{"Tool: FuSaOps-" + fusaops.Version},
		},
	}
	for i, p := range a.Packages {
		// Index-based SPDXID guarantees a syntactically valid identifier
		// regardless of characters in the package name.
		id := fmt.Sprintf("SPDXRef-Package-%d", i)
		doc.Packages = append(doc.Packages, spdxPackage{
			Name:             p.Name,
			SPDXID:           id,
			VersionInfo:      p.Version,
			DownloadLocation: "NOASSERTION",
			FilesAnalyzed:    false,
		})
		doc.Relationships = append(doc.Relationships, spdxRelationship{
			SPDXElementID:      "SPDXRef-DOCUMENT",
			RelatedSPDXElement: id,
			RelationshipType:   "DESCRIBES",
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("sbom: spdx encode: %w", err)
	}
	return nil
}
