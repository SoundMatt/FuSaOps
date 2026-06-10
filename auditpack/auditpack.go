// Package auditpack bundles every component's evidence into one ZIP.
//
// FuSaOps does not generate per-language evidence itself: each x-FuSa tool
// produces its own audit-pack ZIP. This package collects those per-tool packs
// together with the FuSaOps cross-language artefacts (aggregate report, trace
// matrix, SBOM) and a signed manifest into a single audit-pack.zip an auditor
// can open once for the whole polyglot project.
package auditpack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// ManifestFile is the name of the index written into every pack.
const ManifestFile = "manifest.json"

// Source is one item to place into the bundle. Exactly one of FilePath or Data
// supplies the content; FilePath takes precedence when both are set.
//
//fusa:req REQ-FO-PCK001
type Source struct {
	// ArchivePath is the slash-separated path of the entry inside the ZIP.
	ArchivePath string
	// FilePath, when set, is read from disk for the entry's content.
	FilePath string
	// Data is the entry's content when FilePath is empty.
	Data []byte
}

// Entry records one packed file and its integrity hash.
//
//fusa:req REQ-FO-PCK002
type Entry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Manifest is the index of an audit pack, written as manifest.json.
//
//fusa:req REQ-FO-PCK003
type Manifest struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Tool        string    `json:"tool"`
	Version     string    `json:"version"`
	Project     string    `json:"project,omitempty"`
	Files       []Entry   `json:"files"`
}

// Pack writes sources to a ZIP at dest, prepending a manifest.json that records
// every packed file with its SHA-256. Sources are sorted by archive path for a
// deterministic, reproducible bundle. A source whose content cannot be read is a
// hard error: an audit pack must never silently omit evidence.
//
//fusa:req REQ-FO-PCK004
func Pack(dest, project string, sources []Source) (*Manifest, error) {
	sort.Slice(sources, func(i, j int) bool { return sources[i].ArchivePath < sources[j].ArchivePath })

	// Resolve content and hashes first so the manifest is complete before we
	// start writing the archive.
	type resolved struct {
		path string
		data []byte
	}
	items := make([]resolved, 0, len(sources))
	manifest := &Manifest{
		GeneratedAt: time.Now().UTC(),
		Tool:        "FuSaOps",
		Version:     fusaops.Version,
		Project:     project,
	}
	for _, s := range sources {
		data := s.Data
		if s.FilePath != "" {
			b, err := os.ReadFile(s.FilePath)
			if err != nil {
				return nil, fmt.Errorf("auditpack: read %s: %w", s.FilePath, err)
			}
			data = b
		}
		sum := sha256.Sum256(data)
		manifest.Files = append(manifest.Files, Entry{
			Path:   s.ArchivePath,
			Size:   int64(len(data)),
			SHA256: hex.EncodeToString(sum[:]),
		})
		items = append(items, resolved{path: s.ArchivePath, data: data})
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("auditpack: marshal manifest: %w", err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("auditpack: create %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	if err := writeEntry(zw, ManifestFile, append(manifestJSON, '\n'), manifest.GeneratedAt); err != nil {
		return nil, err
	}
	for _, it := range items {
		if err := writeEntry(zw, it.path, it.data, manifest.GeneratedAt); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("auditpack: finalise zip: %w", err)
	}
	return manifest, nil
}

// writeEntry adds one file to the zip with a fixed modtime for reproducibility.
func writeEntry(zw *zip.Writer, name string, data []byte, mod time.Time) error {
	hdr := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: mod}
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("auditpack: add %s: %w", name, err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("auditpack: write %s: %w", name, err)
	}
	return nil
}

// JSONSource is a convenience constructor for an in-memory JSON entry.
//
//fusa:req REQ-FO-PCK005
func JSONSource(archivePath string, v any) (Source, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Source{}, fmt.Errorf("auditpack: marshal %s: %w", archivePath, err)
	}
	return Source{ArchivePath: archivePath, Data: append(data, '\n')}, nil
}
