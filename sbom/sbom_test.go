package sbom

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleDoc = `{
  "module": "github.com/example/m",
  "goVersion": "go1.22",
  "components": [
    {"name":"golang.org/x/sys","version":"v0.1.0","hash":"h1:abc"},
    {"name":"golang.org/x/text","version":"v0.2.0"}
  ]
}`

//fusa:test REQ-FO-SBM001
//fusa:test REQ-FO-SBM002
func TestDocumentDecode(t *testing.T) {
	var d Document
	if err := json.Unmarshal([]byte(sampleDoc), &d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Module != "github.com/example/m" || len(d.Components) != 2 {
		t.Errorf("decode wrong: %+v", d)
	}
	if d.Components[0].Name != "golang.org/x/sys" || d.Components[0].Hash != "h1:abc" {
		t.Errorf("package decode wrong: %+v", d.Components[0])
	}
}

//fusa:test REQ-FO-SBM003
//fusa:test REQ-FO-SBM004
//fusa:test REQ-FO-SBM005
func TestNewMergesAndDedups(t *testing.T) {
	a := New("/r", "proj", []ComponentSBOM{
		{Tool: "gofusa", Language: "go", Available: true, Module: "m1", Packages: []Package{
			{Name: "shared", Version: "v1.0.0"},
			{Name: "goonly", Version: "v0.1.0"},
		}},
		{Tool: "cfusa", Language: "c", Available: true, Module: "m2", Packages: []Package{
			{Name: "shared", Version: "v1.0.0"}, // duplicate name+version → merged once
			{Name: "conly", Version: "v3.0.0"},
		}},
		{Tool: "cpfusa", Language: "cpp", Available: false, Skipped: "not installed"},
	})
	if a.Components[0].Tool != "cfusa" {
		t.Errorf("components not sorted by tool: %s first", a.Components[0].Tool)
	}
	if a.TotalPackages != 3 {
		t.Errorf("dedup failed: got %d packages, want 3", a.TotalPackages)
	}
	// merged packages sorted by name; first is "conly"
	if a.Packages[0].Name != "conly" {
		t.Errorf("packages not sorted: %+v", a.Packages)
	}
	// "shared" keeps the language of the first contributing component in sorted
	// order — cfusa (c) sorts before gofusa (go), so the merge is deterministic.
	for _, p := range a.Packages {
		if p.Name == "shared" && p.Language != "c" {
			t.Errorf("shared language should be c (cfusa sorts first), got %q", p.Language)
		}
	}
}

func sampleAgg() *Aggregate {
	return New("/r", "proj", []ComponentSBOM{
		{Tool: "gofusa", Language: "go", Available: true, Module: "m1", Packages: []Package{
			{Name: "golang.org/x/sys", Version: "v0.1.0"},
		}},
		{Tool: "cfusa", Language: "c", Available: false, Skipped: "cfusa binary not found"},
	})
}

//fusa:test REQ-FO-SBM006
//fusa:test REQ-FO-SBM007
func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAgg(), "json"); err != nil {
		t.Fatal(err)
	}
	var back Aggregate
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if back.TotalPackages != 1 {
		t.Errorf("json lost packages: %+v", back)
	}
}

//fusa:test REQ-FO-SBM008
func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAgg(), "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Aggregate SBOM", "gofusa", "skipped", "golang.org/x/sys"} {
		if !strings.Contains(out, want) {
			t.Errorf("text missing %q\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-SBM009
func TestRenderSPDX(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAgg(), "spdx"); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("spdx not valid json: %v", err)
	}
	if doc["spdxVersion"] != "SPDX-2.3" || doc["dataLicense"] != "CC0-1.0" {
		t.Errorf("spdx header wrong: %v / %v", doc["spdxVersion"], doc["dataLicense"])
	}
	pkgs, _ := doc["packages"].([]any)
	rels, _ := doc["relationships"].([]any)
	if len(pkgs) != 1 || len(rels) != 1 {
		t.Errorf("spdx should describe 1 package: pkgs=%d rels=%d", len(pkgs), len(rels))
	}
	p, ok := pkgs[0].(map[string]any)
	if !ok {
		t.Fatalf("package not an object: %v", pkgs[0])
	}
	id, ok := p["SPDXID"].(string)
	if !ok || !strings.HasPrefix(id, "SPDXRef-Package-") {
		t.Errorf("invalid SPDXID: %v", p["SPDXID"])
	}
}

//fusa:test REQ-FO-SBM006
func TestRenderUnknownAndToFile(t *testing.T) {
	if err := Render(&bytes.Buffer{}, sampleAgg(), "xml"); err == nil {
		t.Error("expected error for unknown format")
	}
	path := filepath.Join(t.TempDir(), "sbom.json")
	if err := RenderToFile(sampleAgg(), "json", path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
	if err := RenderToFile(sampleAgg(), "spdx", filepath.Join(t.TempDir(), "missing", "x")); err == nil {
		t.Error("expected error creating file in missing dir")
	}
}

//fusa:test REQ-FO-SBM009
func TestRenderSPDXNoProject(t *testing.T) {
	var buf bytes.Buffer
	a := New("/r", "", nil) // no project, no packages
	if err := Render(&buf, a, "spdx"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "fusaops-project") {
		t.Error("spdx should fall back to default name")
	}
}

//fusa:test REQ-FO-SBM010
func TestRenderHTML(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAgg(), "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!doctype html>", "Software Bill of Materials", "gofusa", "golang.org/x/sys", "skipped"} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

//fusa:test REQ-FO-SBM010
func TestRenderHTMLNoPackages(t *testing.T) {
	var buf bytes.Buffer
	a := New("/r", "myproj", nil)
	if err := Render(&buf, a, "html"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "myproj") {
		t.Error("html should include project name")
	}
}
