package badge

import (
	"bytes"
	"strings"
	"testing"
)

//fusa:test REQ-FO-BADGE001
func TestNewPass(t *testing.T) {
	b := New(0, 0, "1.41.0")
	if b.Status != StatusPass {
		t.Errorf("want StatusPass, got %d", b.Status)
	}
}

//fusa:test REQ-FO-BADGE001
func TestNewWarn(t *testing.T) {
	b := New(0, 3, "1.41.0")
	if b.Status != StatusWarn {
		t.Errorf("want StatusWarn, got %d", b.Status)
	}
}

//fusa:test REQ-FO-BADGE001
func TestNewFail(t *testing.T) {
	b := New(2, 1, "1.41.0")
	if b.Status != StatusFail {
		t.Errorf("want StatusFail, got %d", b.Status)
	}
}

//fusa:test REQ-FO-BADGE002
func TestRenderPass(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, New(0, 0, "1.41.0")); err != nil {
		t.Fatalf("Render: %v", err)
	}
	svg := buf.String()
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "fusaops") {
		t.Errorf("SVG missing expected content: %q", svg[:min(len(svg), 200)])
	}
	if !strings.Contains(svg, "passing") {
		t.Errorf("pass badge missing 'passing': %q", svg)
	}
}

//fusa:test REQ-FO-BADGE002
func TestRenderWarn(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, New(0, 5, "1.41.0")); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "warnings") {
		t.Errorf("warn badge missing 'warnings': %q", buf.String())
	}
}

//fusa:test REQ-FO-BADGE002
func TestRenderFail(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, New(3, 0, "1.41.0")); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "failing") {
		t.Errorf("fail badge missing 'failing': %q", buf.String())
	}
}
