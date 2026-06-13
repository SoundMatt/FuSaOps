package report

import (
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// renderText writes a human-readable plain-text aggregate report.
//
//fusa:req REQ-FO-RPT010
//fusa:req REQ-FO-RPT017
//fusa:req REQ-FO-RPT019
func renderText(w io.Writer, r *AggregateReport, opts RenderOptions) error {
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
			printTextFindingOpts(w, f, "", opts.ShowFingerprints)
		}
		if opts.ShowSuppressed && len(c.SuppressedFindings) > 0 {
			fmt.Fprintf(w, "  --- suppressed (%d) ---\n", len(c.SuppressedFindings))
			for _, f := range c.SuppressedFindings {
				printTextFindingOpts(w, f, "[SUPPRESSED] ", opts.ShowFingerprints)
			}
		} else if !opts.ShowSuppressed && len(c.SuppressedFindings) > 0 {
			fmt.Fprintf(w, "  (%d suppressed — use --show-suppressed to view)\n", len(c.SuppressedFindings))
		}
		fmt.Fprintln(w)
	}

	if r.Suppressed > 0 {
		fmt.Fprintf(w, "TOTAL: %s — %d findings across %d component(s): %d errors, %d warnings, %d infos (%d suppressed)\n",
			r.Summary.Status(), r.Summary.Total, len(r.Components),
			r.Summary.Errors, r.Summary.Warnings, r.Summary.Infos, r.Suppressed)
	} else {
		fmt.Fprintf(w, "TOTAL: %s — %d findings across %d component(s): %d errors, %d warnings, %d infos\n",
			r.Summary.Status(), r.Summary.Total, len(r.Components),
			r.Summary.Errors, r.Summary.Warnings, r.Summary.Infos)
	}
	return nil
}

func printTextFindingOpts(w io.Writer, f fusaops.Finding, prefix string, showFP bool) {
	fmt.Fprintf(w, "  %s[%s] %-10s", prefix, f.Severity, f.RuleID)
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
	if showFP && f.Fingerprint != "" {
		fmt.Fprintf(w, "    fingerprint: %s\n", f.Fingerprint)
		fmt.Fprintf(w, "    $ fusaops suppress add --fingerprint %s --reason \"\"\n", f.Fingerprint)
	}
}
