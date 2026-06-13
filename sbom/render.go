package sbom

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Render writes the aggregate SBOM to w in the requested format.
//
//fusa:req REQ-FO-SBM006
//fusa:req REQ-FO-SBM010
//fusa:req REQ-FO-SBM011
func Render(w io.Writer, a *Aggregate, format string) error {
	switch format {
	case "", "json":
		return renderJSON(w, a)
	case "text":
		return renderText(w, a)
	case "spdx":
		return renderSPDX(w, a)
	case "html":
		return renderHTML(w, a)
	case "markdown", "md":
		return renderMarkdown(w, a)
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

// renderHTML writes a self-contained HTML SBOM dashboard.
//
//fusa:req REQ-FO-SBM010
func renderHTML(w io.Writer, a *Aggregate) error {
	if err := sbomTemplate.Execute(w, a); err != nil {
		return fmt.Errorf("sbom: html render: %w", err)
	}
	return nil
}

// sbomTemplate is a self-contained, dependency-free SBOM viewer.
var sbomTemplate = template.Must(template.New("sbom").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FuSaOps — SBOM{{if .Project}}: {{.Project}}{{end}}</title>
<style>
 body{font:15px/1.5 system-ui,sans-serif;margin:0;background:#0f1115;color:#e6e6e6}
 header{padding:1.2rem 1.6rem;background:#171a21;border-bottom:1px solid #272b34}
 h1{margin:0;font-size:1.25rem} .meta{color:#9aa3b2;font-size:.85rem;margin-top:.3rem}
 main{padding:1.6rem;max-width:1000px;margin:0 auto}
 h2{font-size:1rem;color:#9aa3b2;margin:1.4rem 0 .4rem}
 table{width:100%;border-collapse:collapse;background:#171a21;border-radius:.6rem;overflow:hidden}
 th,td{padding:.55rem .8rem;text-align:left;border-bottom:1px solid #272b34;font-size:.9rem}
 th{background:#1d2129;color:#9aa3b2;font-weight:600}
 .skip{color:#9aa3b2;font-style:italic} .ver{color:#9aa3b2;font-size:.85rem}
 .lang{display:inline-block;padding:.1rem .45rem;border-radius:.35rem;font-size:.8rem;background:#1d2129;color:#9aa3b2}
</style></head><body>
<header>
 <h1>FuSaOps — Software Bill of Materials{{if .Project}}: {{.Project}}{{end}}</h1>
 <div class="meta">Generated {{.GeneratedAt.Format "2006-01-02 15:04 MST"}} · {{.TotalPackages}} package(s) across {{len .Components}} component(s)</div>
</header>
<main>
 <h2>Components</h2>
 <table>
  <thead><tr><th>Tool</th><th>Language</th><th>Module</th><th class="num">Packages</th></tr></thead>
  <tbody>
  {{range .Components}}
   <tr>
    <td>{{.Tool}}</td><td>{{.Language}}</td>
    {{if .Skipped}}
     <td colspan="2" class="skip">skipped — {{.Skipped}}</td>
    {{else}}
     <td>{{.Module}}</td><td>{{len .Packages}}</td>
    {{end}}
   </tr>
  {{end}}
  </tbody>
 </table>
 <h2>Packages ({{.TotalPackages}} total, de-duplicated)</h2>
 <table>
  <thead><tr><th>Name</th><th>Version</th><th>Language</th></tr></thead>
  <tbody>
  {{range .Packages}}
   <tr>
    <td>{{.Name}}</td>
    <td class="ver">{{if .Version}}{{.Version}}{{else}}—{{end}}</td>
    <td>{{if .Language}}<span class="lang">{{.Language}}</span>{{end}}</td>
   </tr>
  {{end}}
  </tbody>
 </table>
</main></body></html>
`))

// renderMarkdown writes a GFM markdown SBOM summary to w.
//
//fusa:req REQ-FO-SBM011
func renderMarkdown(w io.Writer, a *Aggregate) error {
	fmt.Fprintf(w, "# FuSaOps — Software Bill of Materials")
	if a.Project != "" {
		fmt.Fprintf(w, ": %s", a.Project)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Generated %s · %d package(s) across %d component(s)\n\n",
		a.GeneratedAt.Format("2006-01-02 15:04 MST"), a.TotalPackages, len(a.Components))
	fmt.Fprintln(w, "## Components")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Tool | Language | Module | Packages |")
	fmt.Fprintln(w, "|---|---|---|---:|")
	for _, c := range a.Components {
		if c.Skipped != "" {
			fmt.Fprintf(w, "| %s | %s | _skipped — %s_ | |\n", c.Tool, c.Language, c.Skipped)
		} else {
			fmt.Fprintf(w, "| %s | %s | %s | %d |\n", c.Tool, c.Language, c.Module, len(c.Packages))
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "## Packages (%d total, de-duplicated)\n\n", a.TotalPackages)
	fmt.Fprintln(w, "| Name | Version | Language |")
	fmt.Fprintln(w, "|---|---|---|")
	for _, p := range a.Packages {
		ver := p.Version
		if ver == "" {
			ver = "—"
		}
		name := strings.ReplaceAll(p.Name, "|", "\\|")
		fmt.Fprintf(w, "| %s | %s | %s |\n", name, ver, p.Language)
	}
	return nil
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
