package adapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// argVal returns the value following flag in args, or "".
func argVal(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// capAdapter builds a cmdAdapter wired to a fake runner for capability tests.
func capAdapter(run runnerFunc) *cmdAdapter {
	return &cmdAdapter{name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa", extensions: []string{".go"}, run: run}
}

const traceJSON = `{"requirements":[{"id":"R1"}],"coverage":{"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}}`

//fusa:test REQ-FO-ADP013
//fusa:test REQ-FO-ADP014
func TestAdapterTrace(t *testing.T) {
	// Wrap JSON in incidental output to exercise extractJSON.
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] != "trace" {
			t.Errorf("unexpected subcommand %v", args)
		}
		return []byte("running trace...\n" + traceJSON + "\n"), nil
	})
	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Requirements) != 1 || m.Coverage.TestedRequirements != 1 {
		t.Errorf("trace matrix wrong: %+v", m)
	}

	// Confirm it satisfies the Tracer capability interface.
	var _ Tracer = a
}

func TestAdapterTraceErrors(t *testing.T) {
	boom := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) {
		return nil, errors.New("exec failed")
	})
	if _, err := boom.Trace(context.Background(), "/r"); err == nil {
		t.Error("expected run error")
	}
	bad := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte("no json here"), nil
	})
	if _, err := bad.Trace(context.Background(), "/r"); err == nil {
		t.Error("expected decode error")
	}
}

//fusa:test REQ-FO-ADP015
func TestAdapterQualify(t *testing.T) {
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		out := argVal(args, "--output")
		if out == "" {
			t.Fatal("qualify needs --output")
		}
		return nil, os.WriteFile(out, []byte(`{"total":5,"passed":5,"failed":0}`), 0o600)
	})
	q, err := a.Qualify(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if q.Total != 5 || q.Passed != 5 {
		t.Errorf("qualify wrong: %+v", q)
	}
	var _ Qualifier = a
}

func TestAdapterQualifyErrors(t *testing.T) {
	runErr := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if _, err := runErr.Qualify(context.Background(), "/r"); err == nil {
		t.Error("expected run error")
	}
	// Runner succeeds but writes no file → read error.
	noFile := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil })
	if _, err := noFile.Qualify(context.Background(), "/r"); err == nil {
		t.Error("expected read error when report not written")
	}
}

//fusa:test REQ-FO-ADP016
func TestAdapterSBOM(t *testing.T) {
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		dir := argVal(args, "--output-dir")
		if dir == "" {
			t.Fatal("sbom needs --output-dir")
		}
		doc := `{"module":"m","goVersion":"go1.22","components":[{"name":"x","version":"v1"}]}`
		return nil, os.WriteFile(filepath.Join(dir, "sbom.json"), []byte(doc), 0o600)
	})
	d, err := a.SBOM(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if d.Module != "m" || len(d.Components) != 1 {
		t.Errorf("sbom wrong: %+v", d)
	}
	var _ SBOMer = a
}

func TestAdapterSBOMErrors(t *testing.T) {
	runErr := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if _, err := runErr.SBOM(context.Background(), "/r"); err == nil {
		t.Error("expected run error")
	}
	noFile := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil })
	if _, err := noFile.SBOM(context.Background(), "/r"); err == nil {
		t.Error("expected read error when sbom.json absent")
	}
}

//fusa:test REQ-FO-ADP017
func TestAdapterAuditPack(t *testing.T) {
	a := capAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		out := argVal(args, "--output")
		return nil, os.WriteFile(out, []byte("PK\x03\x04"), 0o600)
	})
	dest := filepath.Join(t.TempDir(), "p.zip")
	if err := a.AuditPack(context.Background(), "/r", dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("pack not written: %v", err)
	}
	var _ Packer = a

	// Runner succeeds but writes nothing → AuditPack must report the gap.
	silent := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil })
	if err := silent.AuditPack(context.Background(), "/r", filepath.Join(t.TempDir(), "none.zip")); err == nil {
		t.Error("expected error when no pack produced")
	}
	// Runner errors.
	boom := capAdapter(func(context.Context, string, string, ...string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if err := boom.AuditPack(context.Background(), "/r", dest); err == nil {
		t.Error("expected run error")
	}
}

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		"prefix {\"a\":1} suffix": `{"a":1}`,
		`{"a":1}`:                 `{"a":1}`,
		"no json at all":          "no json at all",
	}
	for in, want := range cases {
		if got := string(extractJSON([]byte(in))); got != want {
			t.Errorf("extractJSON(%q) = %q, want %q", in, got, want)
		}
	}
}
