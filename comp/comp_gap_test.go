package comp_test

// Gap tests for comp/render.go: the DAL string branch (line 44-45) and the
// ExceedsThreshold loop body (line 53-55) in renderText.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/comp"
)

// TestRenderTextWithDALAndViolation verifies renderText includes the DAL string
// suffix when Report.DAL is non-empty (render.go:44-45) and prints the
// exceeding-threshold line for Functions where ExceedsThreshold is true
// (render.go:53-55).
//
//fusa:test REQ-FO-COMP003
func TestRenderTextWithDALAndViolation(t *testing.T) {
	agg := comp.New("/root", "testproj", []comp.ComponentComp{
		{
			Language: "go",
			Tool:     "gofusa",
			Available: true,
			Report: &comp.Report{
				DAL:            "DAL-A",
				Threshold:      10,
				TotalFunctions: 2,
				Violations:     1,
				Results: []comp.Function{
					{Name: "BigFn", File: "main.go", Line: 5, Complexity: 25, ExceedsThreshold: true},
					{Name: "SmallFn", File: "util.go", Line: 1, Complexity: 3, ExceedsThreshold: false},
				},
			},
		},
	})
	var buf bytes.Buffer
	if err := comp.Render(&buf, agg, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "DAL-A") {
		t.Errorf("renderText: expected DAL-A in output:\n%s", out)
	}
	if !strings.Contains(out, "BigFn") {
		t.Errorf("renderText: expected BigFn (exceeds threshold) in output:\n%s", out)
	}
}
