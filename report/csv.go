package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// renderCSV writes the aggregate report as CSV for spreadsheet consumers.
//
// Columns: language, tool, ruleId, severity, message, file, line, column,
// category, fingerprint. One row per finding across all components. Components
// with no findings and skipped components are omitted; they appear in the JSON
// and text renderers instead.
//
//fusa:req REQ-FO-RPT014
func renderCSV(w io.Writer, r *AggregateReport) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"language", "tool", "ruleId", "severity",
		"message", "file", "line", "column",
		"category", "fingerprint",
	}); err != nil {
		return fmt.Errorf("report: csv header: %w", err)
	}
	for _, c := range r.Components {
		for _, f := range c.Findings {
			row := csvRow(f)
			if err := cw.Write(row); err != nil {
				return fmt.Errorf("report: csv write: %w", err)
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

func csvRow(f fusaops.Finding) []string {
	line := ""
	if f.Location.Line > 0 {
		line = strconv.Itoa(f.Location.Line)
	}
	col := ""
	if f.Location.Column > 0 {
		col = strconv.Itoa(f.Location.Column)
	}
	return []string{
		string(f.Language),
		f.Tool,
		f.QualifiedRuleID(),
		string(f.Severity),
		f.Message,
		f.Location.File,
		line,
		col,
		f.Category,
		f.Fingerprint,
	}
}
