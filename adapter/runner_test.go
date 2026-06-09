package adapter

import (
	"context"
	"runtime"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// exitCmd returns a platform-appropriate command that exits with the given code.
func exitCmd(code int) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "exit", itoa(code)}
	}
	if code == 0 {
		return "true", nil
	}
	return "false", nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return "1"
}

//fusa:test REQ-FO-ADP001
//fusa:test REQ-FO-ADP010
func TestAdapterGetters(t *testing.T) {
	a := newGoFuSa()
	if a.Name() != "go-FuSa" || a.Tool() != "gofusa" || a.Language() != fusaops.LangGo {
		t.Errorf("getters wrong: %s/%s/%s", a.Name(), a.Tool(), a.Language())
	}
}

//fusa:test REQ-FO-ADP003
func TestAvailableUnknownTool(t *testing.T) {
	a := &cmdAdapter{tool: "definitely-not-a-real-binary-xyz", run: defaultRunner}
	if a.Available() {
		t.Error("nonexistent binary should report unavailable")
	}
}

//fusa:test REQ-FO-ADP005
func TestDefaultRunnerExitCodes(t *testing.T) {
	// Exit 0 and exit 1 must both return without a runner error: a non-zero
	// exit means "findings exist", not "failed to run".
	name0, args0 := exitCmd(0)
	if _, err := defaultRunner(context.Background(), t.TempDir(), name0, args0...); err != nil {
		t.Errorf("exit 0: unexpected err %v", err)
	}
	name1, args1 := exitCmd(1)
	if _, err := defaultRunner(context.Background(), t.TempDir(), name1, args1...); err != nil {
		t.Errorf("exit 1: should be swallowed, got %v", err)
	}
}

func TestDefaultRunnerMissingBinary(t *testing.T) {
	if _, err := defaultRunner(context.Background(), t.TempDir(), "no-such-binary-xyz-123"); err == nil {
		t.Error("missing binary should return an error")
	}
}
