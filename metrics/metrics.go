// Package metrics tracks FuSaOps project safety metrics over time.
//
// Use Collect to take a snapshot of current project metrics, Append to add
// it to the time series, and Save to persist it to .fusaops-metrics.json.
//
// Usage:
//
//	ts, err := metrics.Load(projectRoot)
//	snap, err := metrics.Collect(projectRoot)
//	ts = metrics.Append(ts, snap)
//	err = metrics.Save(projectRoot, ts)
package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// MetricsFile is the default filename for the metrics time series.
const MetricsFile = ".fusaops-metrics.json"

// Snapshot captures a point-in-time set of project safety metrics.
//
//fusa:req REQ-FO-MET001
type Snapshot struct {
	Timestamp         time.Time `json:"timestamp"`
	ErrorCount        int       `json:"errorCount"`
	WarningCount      int       `json:"warningCount"`
	InfoCount         int       `json:"infoCount"`
	TotalRequirements int       `json:"totalRequirements"`
	CoveragePct       float64   `json:"coveragePct,omitempty"`
}

// TimeSeries is the full metrics history for a project.
//
//fusa:req REQ-FO-MET001
type TimeSeries struct {
	Project   string     `json:"project"`
	Snapshots []Snapshot `json:"snapshots"`
}

// Load reads the metrics time series from projectRoot.
// If the file does not exist it returns an empty series with no error.
//
//fusa:req REQ-FO-MET002
func Load(projectRoot string) (*TimeSeries, error) {
	path := filepath.Join(projectRoot, MetricsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TimeSeries{}, nil
		}
		return nil, fmt.Errorf("metrics: read %s: %w", MetricsFile, err)
	}
	var ts TimeSeries
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, fmt.Errorf("metrics: parse %s: %w", MetricsFile, err)
	}
	return &ts, nil
}

// Save writes ts to .fusaops-metrics.json in projectRoot.
//
//fusa:req REQ-FO-MET002
func Save(projectRoot string, ts *TimeSeries) error {
	path := filepath.Join(projectRoot, MetricsFile)
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Errorf("metrics: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("metrics: write %s: %w", MetricsFile, err)
	}
	return nil
}

// Append adds snap to ts and returns the updated series.
//
//fusa:req REQ-FO-MET002
func Append(ts *TimeSeries, snap Snapshot) *TimeSeries {
	ts.Snapshots = append(ts.Snapshots, snap)
	return ts
}

// checkReport is a minimal struct for parsing a fusaops check JSON report.
type checkReport struct {
	Components []struct {
		Findings []struct {
			Severity string `json:"severity"`
		} `json:"findings"`
	} `json:"components"`
}

// reqsFile is a minimal struct for parsing .fusa-reqs.json.
type reqsFile struct {
	Requirements []struct{} `json:"requirements"`
}

// coverageReport is a minimal struct for parsing a coverage JSON report.
type coverageReport struct {
	StmtPct float64 `json:"stmtPct"`
}

// Collect reads project artefacts and builds a metrics snapshot.
// It reads check-report.json (or aggregate-report.json), .fusa-reqs.json,
// and coverage-report.json or coverage.out.
//
//fusa:req REQ-FO-MET002
func Collect(projectRoot string) (Snapshot, error) {
	snap := Snapshot{Timestamp: time.Now().UTC()}

	// Parse check/aggregate report JSON for finding counts.
	for _, name := range []string{"check-report.json", "aggregate-report.json"} {
		checkPath := filepath.Join(projectRoot, name)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			continue
		}
		var rep checkReport
		if err := json.Unmarshal(data, &rep); err == nil {
			for _, comp := range rep.Components {
				for _, f := range comp.Findings {
					switch f.Severity {
					case "ERROR":
						snap.ErrorCount++
					case "WARNING":
						snap.WarningCount++
					case "INFO":
						snap.InfoCount++
					}
				}
			}
			break
		}
	}

	// Parse .fusa-reqs.json for requirement count.
	reqPath := filepath.Join(projectRoot, ".fusa-reqs.json")
	if data, err := os.ReadFile(reqPath); err == nil {
		var reqs reqsFile
		if err := json.Unmarshal(data, &reqs); err == nil {
			snap.TotalRequirements = len(reqs.Requirements)
		}
	}

	// Parse coverage report JSON, or fall back to coverage.out.
	covReportPath := filepath.Join(projectRoot, "coverage-report.json")
	if data, err := os.ReadFile(covReportPath); err == nil {
		var cov coverageReport
		if err := json.Unmarshal(data, &cov); err == nil {
			snap.CoveragePct = cov.StmtPct
		}
	}

	return snap, nil
}

// Render writes the time series to w in the requested format ("text" or "json").
//
//fusa:req REQ-FO-MET003
func Render(w io.Writer, ts *TimeSeries, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ts)
	case "text":
		return renderText(w, ts)
	default:
		return fmt.Errorf("metrics: unsupported format %q", format)
	}
}

func renderText(w io.Writer, ts *TimeSeries) error {
	project := ts.Project
	if project == "" {
		project = "(project)"
	}
	fmt.Fprintf(w, "FuSaOps Metrics — %s\n\n", project)
	if len(ts.Snapshots) == 0 {
		fmt.Fprintln(w, "No snapshots recorded. Run 'fusaops metrics record' to record one.")
		return nil
	}
	fmt.Fprintf(w, "%-20s %-6s %-6s %-6s %-5s %s\n",
		"Date", "ERR", "WARN", "INFO", "Reqs", "Coverage%")
	fmt.Fprintln(w, "─────────────────────────────────────────────────────────")
	for _, s := range ts.Snapshots {
		covStr := "n/a"
		if s.CoveragePct > 0 {
			covStr = fmt.Sprintf("%.1f%%", s.CoveragePct)
		}
		fmt.Fprintf(w, "%-20s %-6d %-6d %-6d %-5d %s\n",
			s.Timestamp.Format("2006-01-02 15:04"),
			s.ErrorCount, s.WarningCount, s.InfoCount,
			s.TotalRequirements,
			covStr,
		)
	}
	return nil
}
