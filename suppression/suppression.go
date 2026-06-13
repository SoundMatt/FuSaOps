// Package suppression filters findings that have been acknowledged in a
// .fusaops-suppress.json file. Suppressed findings are removed from the
// active set and counted separately in the aggregate report.
package suppression

import (
	"encoding/json"
	"os"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Suppression is one acknowledged finding entry.
//
//fusa:req REQ-FO-SUP001
type Suppression struct {
	// Fingerprint is the spec §4.2 finding fingerprint to suppress.
	Fingerprint string `json:"fingerprint"`
	// Reason is mandatory human-readable rationale.
	Reason string `json:"reason"`
	// Expires is an optional ISO-8601 date (YYYY-MM-DD) after which the
	// suppression is no longer active and the finding resurfaces.
	Expires string `json:"expires,omitempty"`
}

// Config is the top-level structure of .fusaops-suppress.json.
//
//fusa:req REQ-FO-SUP001
type Config struct {
	Suppressions []Suppression `json:"suppressions"`
}

// LoadConfig reads a suppression config file. Returns an empty Config when
// path is empty (zero-config path).
//
//fusa:req REQ-FO-SUP002
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes cfg to path as indented JSON.
//
//fusa:req REQ-FO-SUP005
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Prune removes expired or invalid-date suppression entries from cfg.
// Returns the pruned config and the number of entries removed.
//
//fusa:req REQ-FO-SUP007
func Prune(cfg Config, now time.Time) (Config, int) {
	var kept []Suppression
	for _, s := range cfg.Suppressions {
		if s.Expires == "" {
			kept = append(kept, s)
			continue
		}
		exp, err := time.Parse("2006-01-02", s.Expires)
		if err != nil || !now.Before(exp.AddDate(0, 0, 1)) {
			continue // expired or malformed: remove
		}
		kept = append(kept, s)
	}
	removed := len(cfg.Suppressions) - len(kept)
	return Config{Suppressions: kept}, removed
}

// Filter partitions findings into kept and suppressed slices.
// A finding is suppressed when its Fingerprint matches a Suppression entry
// that has not yet expired (relative to now). Entries without Fingerprints are
// skipped. Expired entries do not suppress findings.
//
//fusa:req REQ-FO-SUP003
func Filter(findings []fusaops.Finding, cfg Config, now time.Time) (kept, suppressed []fusaops.Finding) {
	active := make(map[string]struct{}, len(cfg.Suppressions))
	for _, s := range cfg.Suppressions {
		if s.Fingerprint == "" {
			continue
		}
		if s.Expires != "" {
			exp, err := time.Parse("2006-01-02", s.Expires)
			if err != nil || !now.Before(exp.AddDate(0, 0, 1)) {
				// malformed date or expired: skip
				continue
			}
		}
		active[s.Fingerprint] = struct{}{}
	}
	for _, f := range findings {
		if f.Fingerprint != "" {
			if _, ok := active[f.Fingerprint]; ok {
				suppressed = append(suppressed, f)
				continue
			}
		}
		kept = append(kept, f)
	}
	return kept, suppressed
}
