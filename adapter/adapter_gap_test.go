package adapter

// Gap tests covering uncovered branches in adapter.go, capabilities.go, and
// cpfusa.go.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// extractOutputArg finds the value following "--output" in args.
func extractOutputArg(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--output" {
			return args[i+1]
		}
	}
	return ""
}

// TestDetectUnreadableSubdir verifies Detect returns an error when the walk
// encounters a directory it cannot read, covering adapter.go:79.17,81.4 and
// adapter.go:97.16,99.3.
//
//fusa:test REQ-FO-ADP004
func TestDetectUnreadableSubdir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics (0o000) not available on Windows")
	}
	root := t.TempDir()
	unreadable := filepath.Join(root, "restricted")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })

	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run:        func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil },
	}
	_, err := a.Detect(root)
	if err == nil {
		t.Error("Detect: expected error for unreadable subdirectory, got nil")
	}
}

// TestCheckReadFileError verifies Check returns an error when the output file
// is removed before it can be read, covering adapter.go:125.16,127.3.
//
//fusa:test REQ-FO-ADP005
func TestCheckReadFileError(t *testing.T) {
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run: func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
			if out := extractOutputArg(args); out != "" {
				_ = os.Remove(out)
			}
			return nil, nil
		},
	}
	_, err := a.Check(context.Background(), t.TempDir())
	if err == nil {
		t.Error("Check: expected error when output file removed by runner")
	}
}

// TestApplicableDetectError verifies Applicable propagates a Detect error,
// covering adapter.go:268.17,270.4.
//
//fusa:test REQ-FO-ADP008
func TestApplicableDetectError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics (0o000) not available on Windows")
	}
	root := t.TempDir()
	unreadable := filepath.Join(root, "restricted")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })

	a := &cmdAdapter{
		name: "test-adapter", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run:        func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil },
	}
	r := NewRegistry()
	r.MustRegister(a)
	_, err := r.Applicable(root)
	if err == nil {
		t.Error("Applicable: expected error when Detect fails on unreadable subdir")
	}
}

// TestQualifyReadFileError verifies Qualify returns an error when the runner
// removes the output file before ReadFile, covering capabilities.go:106.16,108.3.
//
//fusa:test REQ-FO-ADP015
func TestQualifyReadFileError(t *testing.T) {
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if out := extractOutputArg(args); out != "" {
			_ = os.Remove(out)
		}
		return nil, nil
	})
	_, err := a.Qualify(context.Background(), t.TempDir())
	if err == nil {
		t.Error("Qualify: expected error when output file removed by runner")
	}
}

// TestSBOMBadJSON verifies SBOM returns an error when sbom.json contains
// invalid JSON, covering capabilities.go:136.64,138.3.
//
//fusa:test REQ-FO-ADP016
func TestSBOMBadJSON(t *testing.T) {
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		dir := argVal(args, "--output-dir")
		if dir != "" {
			return nil, os.WriteFile(filepath.Join(dir, "sbom.json"), []byte("not valid json"), 0o600)
		}
		return nil, nil
	})
	_, err := a.SBOM(context.Background(), t.TempDir())
	if err == nil {
		t.Error("SBOM: expected error when sbom.json contains invalid JSON")
	}
}

// TestCppFuSaStandardsReadFileError verifies cppFuSaAdapter.Standards returns
// an error when the runner removes the output file before ReadFile, covering
// cpfusa.go:40.16,42.3.
//
//fusa:test REQ-FO-ADP025
func TestCppFuSaStandardsReadFileError(t *testing.T) {
	a := &cppFuSaAdapter{&cmdAdapter{
		name: "cpp-FuSa", language: fusaops.LangCpp, tool: "cpfusa",
		extensions: []string{".cpp"},
		run: func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
			if out := extractOutputArg(args); out != "" {
				_ = os.Remove(out)
			}
			return nil, nil
		},
	}}
	_, err := a.Standards(context.Background(), t.TempDir(), "iso26262")
	if err == nil {
		t.Error("cppFuSaAdapter.Standards: expected error when output file removed by runner")
	}
}
