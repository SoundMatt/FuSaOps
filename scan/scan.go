// Package scan discovers which languages are present in a repository and how
// much source each accounts for. It provides the human-facing "what is in this
// repo" view that precedes a full orchestrated check.
package scan

import (
	"os"
	"path/filepath"
	"sort"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// langExtensions maps each supported language to its source file extensions.
//
//fusa:req REQ-FO-SCAN001
var langExtensions = map[fusaops.Language][]string{
	fusaops.LangGo:     {".go"},
	fusaops.LangC:      {".c"},
	fusaops.LangCpp:    {".cpp", ".cc", ".cxx", ".hpp", ".hh"},
	fusaops.LangRust:   {".rs"},
	fusaops.LangPython: {".py"},
	fusaops.LangJava:   {".java"},
}

// LangStat records how many source files of a language were found.
type LangStat struct {
	Language fusaops.Language `json:"language"`
	Files    int              `json:"files"`
}

// Result is the outcome of scanning a repository.
type Result struct {
	Root  string     `json:"root"`
	Stats []LangStat `json:"stats"`
}

// Languages returns the detected languages in deterministic order.
//
//fusa:req REQ-FO-SCAN002
func (r *Result) Languages() []fusaops.Language {
	out := make([]fusaops.Language, 0, len(r.Stats))
	for _, s := range r.Stats {
		out = append(out, s.Language)
	}
	return out
}

// skipDir reports whether a directory is excluded from scanning.
func skipDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor", "build", "dist",
		".cache", "target", "out", ".idea", ".vscode":
		return true
	}
	return false
}

// Scan walks root and counts source files per language, skipping VCS, build
// and vendor directories. Languages with zero files are omitted. The returned
// stats are sorted by descending file count then language name.
//
//fusa:req REQ-FO-SCAN003
func Scan(root string, exclude ...string) (*Result, error) {
	extLang := make(map[string]fusaops.Language)
	for lang, exts := range langExtensions {
		for _, e := range exts {
			extLang[e] = lang
		}
	}
	excluded := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		excluded[e] = struct{}{}
	}

	counts := make(map[fusaops.Language]int)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				if _, ok := excluded[d.Name()]; ok {
					return filepath.SkipDir
				}
				if skipDir(d.Name()) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if lang, ok := extLang[filepath.Ext(d.Name())]; ok {
			counts[lang]++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	stats := make([]LangStat, 0, len(counts))
	for lang, n := range counts {
		stats = append(stats, LangStat{Language: lang, Files: n})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Files != stats[j].Files {
			return stats[i].Files > stats[j].Files
		}
		return stats[i].Language < stats[j].Language
	})
	return &Result{Root: root, Stats: stats}, nil
}
