package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// renderJSON writes the aggregate report as indented JSON.
//
//fusa:req REQ-FO-RPT009
func renderJSON(w io.Writer, r *AggregateReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("report: json encode: %w", err)
	}
	return nil
}
