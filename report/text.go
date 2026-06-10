package report

import (
	"fmt"
	"io"
)

// renderText writes a human-readable plain-text aggregate report.
//
//fusa:req REQ-FO-RPT010
func renderText(w io.Writer, r *AggregateReport) error {
	fmt.Fprintln(w, "FuSaOps Multi-Language Safety Report")
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	if r.Project != "" {
		fmt.Fprintf(w, "Project:   %s\n", r.Project)
	}
	fmt.Fprintf(w, "Root:      %s\n", r.Root)
	fmt.Fprintf(w, "Status:    %s\n\n", r.Summary.Status())

	for _, c := range r.Components {
		fmt.Fprintf(w, "── %s (%s) ──\n", c.Tool, c.Language)
		if c.Skipped != "" {
			fmt.Fprintf(w, "  skipped: %s\n\n", c.Skipped)
			continue
		}
		fmt.Fprintf(w, "  %s  %d findings (%d errors, %d warnings, %d infos)\n\n",
			c.Summary.Status(), c.Summary.Total, c.Summary.Errors, c.Summary.Warnings, c.Summary.Infos)
		for _, f := range c.Findings {
			fmt.Fprintf(w, "  [%s] %-10s", f.Severity, f.RuleID)
			if f.Category != "" {
				fmt.Fprintf(w, " [%s]", f.Category)
			}
			fmt.Fprintf(w, " %s", f.Message)
			if f.Location.File != "" {
				fmt.Fprintf(w, " (%s", f.Location.File)
				if f.Location.Line > 0 {
					fmt.Fprintf(w, ":%d", f.Location.Line)
				}
				fmt.Fprint(w, ")")
			}
			fmt.Fprintln(w)
			if f.Remediation != "" {
				fmt.Fprintf(w, "    → %s\n", f.Remediation)
			}
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "TOTAL: %s — %d findings across %d component(s): %d errors, %d warnings, %d infos\n",
		r.Summary.Status(), r.Summary.Total, len(r.Components),
		r.Summary.Errors, r.Summary.Warnings, r.Summary.Infos)
	return nil
}
