package vuln

import "testing"

// TestRunCommandGoVersion verifies runCommand can invoke a real binary, covering
// the real exec path used by the production defaultRunner.
//
//fusa:test REQ-FO-VULN002
func TestRunCommandGoVersion(t *testing.T) {
	out, err := runCommand("go", "version")
	if err != nil {
		t.Fatalf("runCommand go version: %v", err)
	}
	if len(out) == 0 {
		t.Error("runCommand: expected non-empty output from go version")
	}
}

// TestDefaultRunnerEmptyArgs verifies defaultRunner returns an error for an
// empty argument list (len(args) == 0 branch).
//
//fusa:test REQ-FO-VULN002
func TestDefaultRunnerEmptyArgs(t *testing.T) {
	if _, err := defaultRunner(); err == nil {
		t.Error("defaultRunner: expected error for empty args")
	}
}

// TestDefaultRunnerGoVersion verifies defaultRunner delegates to runCommand for
// a non-empty arg list, covering the real execution path.
//
//fusa:test REQ-FO-VULN002
func TestDefaultRunnerGoVersion(t *testing.T) {
	out, err := defaultRunner("go", "version")
	if err != nil {
		t.Fatalf("defaultRunner go version: %v", err)
	}
	if len(out) == 0 {
		t.Error("defaultRunner: expected non-empty output from go version")
	}
}
