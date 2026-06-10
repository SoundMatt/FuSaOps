package auditpack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// readZip returns a map of archive path → content for assertions.
func readZip(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = zr.Close() }()
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		out[f.Name] = b
	}
	return out
}

//fusa:test REQ-FO-PCK001
//fusa:test REQ-FO-PCK002
//fusa:test REQ-FO-PCK003
//fusa:test REQ-FO-PCK004
func TestPack(t *testing.T) {
	dir := t.TempDir()
	// a file source on disk
	src := filepath.Join(dir, "evidence.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "audit-pack.zip")

	manifest, err := Pack(dest, "proj", []Source{
		{ArchivePath: "z/last.bin", Data: []byte("zzz")},
		{ArchivePath: "a/first.txt", FilePath: src},
	})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	if manifest.Tool != "FuSaOps" || manifest.Project != "proj" || manifest.Version == "" {
		t.Errorf("manifest metadata wrong: %+v", manifest)
	}
	// Sorted by archive path: a/first.txt before z/last.bin.
	if len(manifest.Files) != 2 || manifest.Files[0].Path != "a/first.txt" {
		t.Fatalf("entries not sorted: %+v", manifest.Files)
	}
	wantHash := sha256.Sum256([]byte("hello"))
	if manifest.Files[0].SHA256 != hex.EncodeToString(wantHash[:]) || manifest.Files[0].Size != 5 {
		t.Errorf("file entry hash/size wrong: %+v", manifest.Files[0])
	}

	contents := readZip(t, dest)
	if _, ok := contents[ManifestFile]; !ok {
		t.Error("manifest.json not in zip")
	}
	if string(contents["a/first.txt"]) != "hello" || string(contents["z/last.bin"]) != "zzz" {
		t.Errorf("zip contents wrong: %v", contents)
	}
	// manifest.json round-trips and matches the returned manifest.
	var back Manifest
	if err := json.Unmarshal(contents[ManifestFile], &back); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	if len(back.Files) != 2 {
		t.Errorf("packed manifest wrong: %+v", back)
	}
}

//fusa:test REQ-FO-PCK004
func TestPackMissingSourceFails(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.zip")
	_, err := Pack(dest, "p", []Source{{ArchivePath: "x", FilePath: "/no/such/file"}})
	if err == nil {
		t.Error("expected hard error when a source cannot be read")
	}
}

//fusa:test REQ-FO-PCK004
func TestPackBadDest(t *testing.T) {
	_, err := Pack(filepath.Join(t.TempDir(), "missing", "out.zip"), "p", nil)
	if err == nil {
		t.Error("expected error creating zip in missing dir")
	}
}

//fusa:test REQ-FO-PCK005
func TestJSONSource(t *testing.T) {
	s, err := JSONSource("report.json", map[string]int{"n": 1})
	if err != nil {
		t.Fatal(err)
	}
	if s.ArchivePath != "report.json" || s.FilePath != "" {
		t.Errorf("source wrong: %+v", s)
	}
	var v map[string]int
	if err := json.Unmarshal(s.Data, &v); err != nil || v["n"] != 1 {
		t.Errorf("json source data wrong: %v %v", v, err)
	}

	if _, err := JSONSource("bad.json", make(chan int)); err == nil {
		t.Error("expected marshal error for unmarshalable value")
	}
}
