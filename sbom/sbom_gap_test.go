package sbom

// Gap tests covering uncovered branches in sbom.go and render.go.

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderToFileEmptyPath verifies RenderToFile writes to stdout (no error)
// when path is empty, covering render.go:41.16,43.3.
//
//fusa:test REQ-FO-SBM006
func TestRenderToFileEmptyPath(t *testing.T) {
	a := New("/r", "p", nil)
	// path="" → writes to os.Stdout; we just check it does not error.
	if err := RenderToFile(a, "text", ""); err != nil {
		t.Errorf("RenderToFile empty path: unexpected error: %v", err)
	}
}

// TestRenderJSONWriteError verifies renderJSON returns an error when the
// writer fails, covering render.go:56.38,58.3.
//
//fusa:test REQ-FO-SBM007
func TestRenderJSONWriteError(t *testing.T) {
	a := New("/r", "p", nil)
	if err := renderJSON(errWriter{}, a); err == nil {
		t.Error("renderJSON: expected error when writer fails")
	}
}

// TestRenderMarkdownEmptyVersion verifies renderMarkdown emits "—" for
// packages with an empty Version, covering render.go:208.16,210.4.
//
//fusa:test REQ-FO-SBM011
func TestRenderMarkdownEmptyVersion(t *testing.T) {
	a := &Aggregate{
		Packages: []Package{
			{Name: "no-version-pkg", Version: "", Language: "go"},
		},
	}
	var buf bytes.Buffer
	if err := renderMarkdown(&buf, a); err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	if !strings.Contains(buf.String(), "—") {
		t.Errorf("renderMarkdown: expected em-dash for empty version:\n%s", buf.String())
	}
}

// TestRenderSPDXWriteError verifies renderSPDX returns an error when the
// writer fails, covering render.go:254.40,256.3.
//
//fusa:test REQ-FO-SBM009
func TestRenderSPDXWriteError(t *testing.T) {
	a := New("/r", "p", nil)
	if err := renderSPDX(errWriter{}, a); err == nil {
		t.Error("renderSPDX: expected error when writer fails")
	}
}

// TestNewSortSameNameDifferentVersion verifies that two packages with the same
// name but different versions are both retained and sorted by version,
// covering sbom.go:88.3,88.47.
//
//fusa:test REQ-FO-SBM004
func TestNewSortSameNameDifferentVersion(t *testing.T) {
	comps := []ComponentSBOM{
		{
			Language:  "go",
			Tool:      "gofusa",
			Available: true,
			Packages: []Package{
				{Name: "pkg-a", Version: "v2.0.0"},
				{Name: "pkg-a", Version: "v1.0.0"},
			},
		},
	}
	a := New("/r", "p", comps)
	if a.TotalPackages != 2 {
		t.Errorf("New: expected 2 packages, got %d", a.TotalPackages)
	}
	if a.Packages[0].Version != "v1.0.0" {
		t.Errorf("New: expected v1.0.0 first (sort by version), got %s", a.Packages[0].Version)
	}
}
