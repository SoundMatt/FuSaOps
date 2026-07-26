package scan

import (
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

//fusa:test REQ-FO-SCAN001
//fusa:test REQ-FO-SCAN002
//fusa:test REQ-FO-SCAN003
func TestScanCountsAndSorts(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "a.go", "b.go", "c.go", "x.cpp", "y.c")
	res, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stats) != 3 {
		t.Fatalf("got %d languages, want 3", len(res.Stats))
	}
	// Go has the most files, must sort first.
	if res.Stats[0].Language != fusaops.LangGo || res.Stats[0].Files != 3 {
		t.Errorf("first stat: %+v", res.Stats[0])
	}
	langs := res.Languages()
	if len(langs) != 3 {
		t.Errorf("Languages(): got %d", len(langs))
	}
}

func TestScanSkipsVendorAndExclude(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "main.go", "vendor/dep.go", "gen/auto.go")
	res, err := Scan(dir, "gen")
	if err != nil {
		t.Fatal(err)
	}
	// vendor (built-in skip) + gen (explicit exclude) dropped → only main.go.
	if len(res.Stats) != 1 || res.Stats[0].Files != 1 {
		t.Errorf("expected 1 go file, got %+v", res.Stats)
	}
}

func TestScanEmpty(t *testing.T) {
	res, err := Scan(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stats) != 0 {
		t.Errorf("expected no stats, got %+v", res.Stats)
	}
}

//fusa:test REQ-FO-SCAN003
func TestScanTraversesNonSkippedSubdir(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "src/main.go") // "src" is not in the skip list → skipDir returns false
	res, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stats) != 1 || res.Stats[0].Files != 1 {
		t.Errorf("expected 1 go file in src/ subdir, got %+v", res.Stats)
	}
}
