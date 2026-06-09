package adapter

import (
	"context"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func TestAdapterGetters(t *testing.T) {
	a := newGoFuSa()
	if a.Name() != "go-FuSa" || a.Tool() != "gofusa" || a.Language() != fusaops.LangGo {
		t.Errorf("getters wrong: %s/%s/%s", a.Name(), a.Tool(), a.Language())
	}
}

func TestAvailableUnknownTool(t *testing.T) {
	a := &cmdAdapter{tool: "definitely-not-a-real-binary-xyz", run: defaultRunner}
	if a.Available() {
		t.Error("nonexistent binary should report unavailable")
	}
}

func TestDefaultRunnerExitCodes(t *testing.T) {
	// "true" exits 0, "false" exits 1; both must return without a runner error
	// (a non-zero exit means "findings exist", not "failed to run").
	if _, err := defaultRunner(context.Background(), t.TempDir(), "true"); err != nil {
		t.Errorf("true: unexpected err %v", err)
	}
	if _, err := defaultRunner(context.Background(), t.TempDir(), "false"); err != nil {
		t.Errorf("false (exit 1): should be swallowed, got %v", err)
	}
}

func TestDefaultRunnerMissingBinary(t *testing.T) {
	if _, err := defaultRunner(context.Background(), t.TempDir(), "no-such-binary-xyz-123"); err == nil {
		t.Error("missing binary should return an error")
	}
}
