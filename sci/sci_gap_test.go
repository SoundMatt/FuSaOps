package sci

// Gap tests covering uncovered branches in sci.go renderText.

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderTextLanguageField verifies that a ConfigItem with a non-empty
// Language field emits "[lang]" in the text output, covering
// sci.go:243.27,245.5.
//
//fusa:test REQ-FO-SCI004
func TestRenderTextLanguageField(t *testing.T) {
	s := &SCI{
		Items: []ConfigItem{
			{
				ID:       "COMP-001",
				Name:     "TestLib",
				Kind:     KindComponent,
				Language: "go",
				Present:  true,
			},
		},
	}
	var buf bytes.Buffer
	if err := renderText(&buf, s); err != nil {
		t.Fatalf("renderText: %v", err)
	}
	if !strings.Contains(buf.String(), "[go]") {
		t.Errorf("renderText: expected [go] in output:\n%s", buf.String())
	}
}

// TestRenderTextSHA256Field verifies that a ConfigItem with a non-empty SHA256
// field emits the "SHA256:" line, covering sci.go:247.25,249.5.
//
//fusa:test REQ-FO-SCI004
func TestRenderTextSHA256Field(t *testing.T) {
	s := &SCI{
		Items: []ConfigItem{
			{
				ID:      "ART-001",
				Name:    "report.zip",
				Kind:    KindArtefact,
				SHA256:  "abc123def456",
				Size:    1024,
				Present: true,
			},
		},
	}
	var buf bytes.Buffer
	if err := renderText(&buf, s); err != nil {
		t.Fatalf("renderText: %v", err)
	}
	if !strings.Contains(buf.String(), "SHA256: abc123def456") {
		t.Errorf("renderText: expected SHA256 line in output:\n%s", buf.String())
	}
}
