package trace

// Gap tests for trace render.go: renderText with Qualification.Failed > 0
// (line 86), renderMarkdown no-title gap entry (line 217), and RenderToFile
// with empty path writing to stdout (line 35).

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderTextQualificationFailed verifies that renderText prints the
// "(N failed)" suffix when Qualification.Failed > 0, covering render.go:86.
//
//fusa:test REQ-FO-TRC014
func TestRenderTextQualificationFailed(t *testing.T) {
	agg := New("/r", "", []ComponentTrace{
		{
			Tool:          "gofusa",
			Language:      "go",
			Available:     true,
			Coverage:      Coverage{TotalRequirements: 4, TracedRequirements: 4, TestedRequirements: 4},
			Qualification: &Qualification{Total: 3, Passed: 2, Failed: 1},
		},
	})
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatalf("renderText with failed qualification: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "1 failed") {
		t.Errorf("renderText should include '1 failed' for Failed=1:\n%s", out)
	}
}

// TestRenderMarkdownNoTitleGap verifies the no-title branch in the per-component
// gap list of renderMarkdown (render.go:217) by including a Requirement with
// an empty Title.
//
//fusa:test REQ-FO-TRC018
func TestRenderMarkdownNoTitleGap(t *testing.T) {
	agg := New("/r", "", []ComponentTrace{
		{
			Tool:      "gofusa",
			Language:  "go",
			Available: true,
			Coverage:  Coverage{TotalRequirements: 2, TracedRequirements: 1, TestedRequirements: 1},
			Requirements: []Requirement{
				{ID: "REQ-NOTITLE-001", Title: ""}, // no title — triggers else branch
			},
		},
	})
	var buf bytes.Buffer
	if err := Render(&buf, agg, "markdown"); err != nil {
		t.Fatalf("renderMarkdown with no-title gap: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "REQ-NOTITLE-001") {
		t.Errorf("renderMarkdown should include gap ID REQ-NOTITLE-001:\n%s", out)
	}
}

// TestRenderToFileEmptyPath verifies RenderToFile delegates to os.Stdout when
// path is empty, covering render.go:35.  Output appears in test log output;
// the test only checks that no error is returned.
//
//fusa:test REQ-FO-TRC012
func TestRenderToFileEmptyPath(t *testing.T) {
	agg := New("/r", "", []ComponentTrace{
		{Tool: "gofusa", Language: "go", Available: true,
			Coverage: Coverage{TotalRequirements: 1, TracedRequirements: 1, TestedRequirements: 1}},
	})
	if err := RenderToFile(agg, "text", ""); err != nil {
		t.Fatalf("RenderToFile with empty path: %v", err)
	}
}
