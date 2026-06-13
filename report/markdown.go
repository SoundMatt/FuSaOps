package report

import (
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// renderMarkdown writes the aggregate report as GitHub-Flavored Markdown.
//
// The output is suitable for pasting into a PR comment, a GitHub/GitLab wiki
// page, or any Markdown-capable document. The structure mirrors the text
// renderer but uses GFM tables and badges so the report renders well in
// browser-rendered Markdown.
//
//fusa:req REQ-FO-RPT015
func renderMarkdown(w io.Writer, r *AggregateReport) error {
	status := r.Summary.Status()
	badge := markdownBadge(status)
	fmt.Fprintf(w, "# FuSaOps Report %s\n\n", badge)
	if r.Project != "" {
		fmt.Fprintf(w, "**Project:** %s  \n", r.Project)
	}
	fmt.Fprintf(w, "**Generated:** %s  \n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "**Root:** `%s`  \n\n", r.Root)

	// Summary table.
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| | Count |\n|---|---|\n")
	fmt.Fprintf(w, "| ❌ Errors | %d |\n", r.Summary.Errors)
	fmt.Fprintf(w, "| ⚠️ Warnings | %d |\n", r.Summary.Warnings)
	fmt.Fprintf(w, "| ℹ️ Info | %d |\n", r.Summary.Infos)
	fmt.Fprintf(w, "| **Total** | **%d** |\n", r.Summary.Total)
	if r.Suppressed > 0 {
		fmt.Fprintf(w, "| 🔕 Suppressed | %d |\n", r.Suppressed)
	}
	fmt.Fprintln(w)

	// Per-component sections.
	fmt.Fprintf(w, "## Components\n\n")
	for _, c := range r.Components {
		icon := "✅"
		if c.Skipped != "" {
			icon = "⏭️"
		} else if c.Summary.Errors > 0 {
			icon = "❌"
		} else if c.Summary.Warnings > 0 {
			icon = "⚠️"
		}
		fmt.Fprintf(w, "### %s `%s / %s`\n\n", icon, c.Language, c.Tool)
		if c.Skipped != "" {
			fmt.Fprintf(w, "_Skipped: %s_\n\n", c.Skipped)
			continue
		}
		if len(c.Findings) == 0 {
			fmt.Fprintf(w, "_No findings._\n\n")
			continue
		}
		fmt.Fprintf(w, "| Severity | Rule | Message | Location |\n|---|---|---|---|\n")
		for _, f := range c.Findings {
			sev := markdownSeverityIcon(f.Severity)
			loc := ""
			if f.Location.File != "" {
				if f.Location.Line > 0 {
					loc = fmt.Sprintf("`%s:%d`", f.Location.File, f.Location.Line)
				} else {
					loc = fmt.Sprintf("`%s`", f.Location.File)
				}
			}
			fmt.Fprintf(w, "| %s | `%s` | %s | %s |\n",
				sev, markdownEscape(f.RuleID), markdownEscape(f.Message), loc)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func markdownBadge(status string) string {
	switch status {
	case "PASS":
		return "![PASS](https://img.shields.io/badge/FuSaOps-PASS-brightgreen)"
	case "WARN":
		return "![WARN](https://img.shields.io/badge/FuSaOps-WARN-yellow)"
	case "FAIL":
		return "![FAIL](https://img.shields.io/badge/FuSaOps-FAIL-red)"
	default:
		return "![PENDING](https://img.shields.io/badge/FuSaOps-PENDING-lightgrey)"
	}
}

func markdownSeverityIcon(s fusaops.Severity) string {
	switch s {
	case fusaops.SeverityError:
		return "❌ ERROR"
	case fusaops.SeverityWarning:
		return "⚠️ WARN"
	default:
		return "ℹ️ INFO"
	}
}

// markdownEscape escapes pipe characters so they don't break GFM tables.
func markdownEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			out = append(out, '\\', '|')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
