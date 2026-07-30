// Package history persists and retrieves fusaops run snapshots so the dashboard
// can render a severity trend across multiple check runs.
//
// Snapshots are written to .fusaops-history.jsonl in the project root — one
// JSON object per line (JSONL), newest entry appended last. Reads return the
// most-recent entries up to the requested limit. At most MaxSnapshots entries
// are retained; older entries are trimmed on the next write.
package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/report"
)

const (
	// Filename is the JSONL file written to the project root.
	Filename = ".fusaops-history.jsonl"

	// MaxSnapshots is the number of entries retained between trims.
	MaxSnapshots = 100
)

// storeMu serialises the non-atomic read-modify-write of the history file so
// concurrent compute paths (startup, scheduler, /refresh) cannot race and lose
// or corrupt snapshots.
var storeMu sync.Mutex

// LanguageSummary records per-language finding counts in a snapshot.
//
//fusa:req REQ-FO-HST001
type LanguageSummary struct {
	Language string `json:"language"`
	Total    int    `json:"total"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
}

// Snapshot records the outcome of one fusaops check run.
//
//fusa:req REQ-FO-HST001
type Snapshot struct {
	RunAt     time.Time         `json:"runAt"`
	Status    string            `json:"status"` // "PASS" or "FAIL"
	Total     int               `json:"total"`
	Errors    int               `json:"errors"`
	Warnings  int               `json:"warnings"`
	Infos     int               `json:"infos"`
	Languages []LanguageSummary `json:"languages"`
}

// FromReport builds a Snapshot from an AggregateReport.
//
//fusa:req REQ-FO-HST001
func FromReport(rep *report.AggregateReport) Snapshot {
	s := Snapshot{
		RunAt:    time.Now().UTC(),
		Status:   "PASS",
		Total:    rep.Summary.Total,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}
	if rep.HasErrors() {
		s.Status = "FAIL"
	}
	for _, c := range rep.Components {
		if len(c.Findings) == 0 {
			continue
		}
		ls := LanguageSummary{Language: string(c.Language)}
		for _, f := range c.Findings {
			ls.Total++
			switch f.Severity {
			case fusaops.SeverityError:
				ls.Errors++
			case fusaops.SeverityWarning:
				ls.Warnings++
			}
		}
		s.Languages = append(s.Languages, ls)
	}
	return s
}

// Store appends snap to the JSONL history file in dir, then trims the file to
// at most MaxSnapshots entries.
//
//fusa:req REQ-FO-HST002
func Store(dir string, snap Snapshot) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	path := filepath.Join(dir, Filename)

	existing, _ := loadAll(path)
	existing = append(existing, snap)
	if len(existing) > MaxSnapshots {
		existing = existing[len(existing)-MaxSnapshots:]
	}
	return writeAll(path, existing)
}

// Load returns the most-recent limit snapshots from dir (oldest first).
// If limit <= 0 all stored snapshots are returned. A missing file is not
// an error; it returns an empty slice.
//
//fusa:req REQ-FO-HST002
func Load(dir string, limit int) ([]Snapshot, error) {
	path := filepath.Join(dir, Filename)
	all, err := loadAll(path)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}

func loadAll(path string) ([]Snapshot, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Snapshot
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var s Snapshot
		if err := json.Unmarshal(line, &s); err == nil {
			out = append(out, s)
		}
	}
	return out, sc.Err()
}

// Prune removes old entries from the JSONL history file in dir, retaining at
// most keep entries (newest). Returns the number of entries removed. A missing
// file is not an error; returns 0, nil. If keep <= 0, MaxSnapshots is used.
//
//fusa:req REQ-FO-HST005
func Prune(dir string, keep int) (int, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	if keep <= 0 {
		keep = MaxSnapshots
	}
	path := filepath.Join(dir, Filename)
	all, err := loadAll(path)
	if err != nil {
		return 0, err
	}
	if len(all) <= keep {
		return 0, nil
	}
	removed := len(all) - keep
	return removed, writeAll(path, all[removed:])
}

// writeAll writes snaps atomically: it encodes to a temp file in the same
// directory (mode 0640, matching the tool's other state files rather than the
// world-readable 0644 of os.Create) then renames it over path, so a crash or a
// concurrent read never observes a truncated history file.
func writeAll(path string, snaps []Snapshot) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".fusaops-history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName) // no-op after a successful rename
	}()
	if err := tmp.Chmod(0o640); err != nil {
		return err
	}
	enc := json.NewEncoder(tmp)
	for _, s := range snaps {
		if err := enc.Encode(s); err != nil {
			return err
		}
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
