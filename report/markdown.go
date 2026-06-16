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
//fusa:req REQ-FO-RPT017
//fusa:req REQ-FO-RPT019
func renderMarkdown(w io.Writer, r *AggregateReport, opts RenderOptions) error {
	status := r.Summary.Status()
	badge := markdownBadge(status)
	fmt.Fprintf(w, "# FuSaOps Report %s\n\n", badge)
	if r.Project != "" {
		fmt.Fprintf(w, "**Project:** %s  \n", r.Project)
	}
	fmt.Fprintf(w, "**Generated:** %s  \n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "**Root:** `%s`  \n", r.Root)
	if r.Standard != "" {
		level := r.ASIL
		if r.SIL != "" {
			level = r.SIL
		} else if r.DAL != "" {
			level = r.DAL
		}
		if level != "" {
			fmt.Fprintf(w, "**Standard:** %s (%s)  \n", r.Standard, level)
		} else {
			fmt.Fprintf(w, "**Standard:** %s  \n", r.Standard)
		}
	}
	fmt.Fprintln(w)

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
		} else {
			if opts.ShowFingerprints {
				fmt.Fprintf(w, "| Severity | Rule | Message | Location | Fingerprint |\n|---|---|---|---|---|\n")
				for _, f := range c.Findings {
					fmt.Fprintf(w, "| %s | `%s` | %s | %s | `%s` |\n",
						markdownSeverityIcon(f.Severity), markdownEscape(f.RuleID),
						markdownEscape(f.Message), markdownLoc(f), markdownEscape(f.Fingerprint))
				}
			} else {
				fmt.Fprintf(w, "| Severity | Rule | Message | Location |\n|---|---|---|---|\n")
				for _, f := range c.Findings {
					fmt.Fprintf(w, "| %s | `%s` | %s | %s |\n",
						markdownSeverityIcon(f.Severity), markdownEscape(f.RuleID),
						markdownEscape(f.Message), markdownLoc(f))
				}
			}
			fmt.Fprintln(w)
		}
		if len(c.SuppressedFindings) > 0 {
			openAttr := ""
			if opts.ShowSuppressed {
				openAttr = " open"
			}
			fmt.Fprintf(w, "<details%s>\n<summary>%d suppressed finding(s)</summary>\n\n", openAttr, len(c.SuppressedFindings))
			fmt.Fprintf(w, "| Severity | Rule | Message | Location |\n|---|---|---|---|\n")
			for _, f := range c.SuppressedFindings {
				fmt.Fprintf(w, "| %s | `%s` | %s | %s |\n",
					markdownSeverityIcon(f.Severity), markdownEscape(f.RuleID),
					markdownEscape(f.Message), markdownLoc(f))
			}
			fmt.Fprintf(w, "\n</details>\n\n")
		}
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

func markdownLoc(f fusaops.Finding) string {
	if f.Location.File == "" {
		return ""
	}
	if f.Location.Line > 0 {
		return fmt.Sprintf("`%s:%d`", f.Location.File, f.Location.Line)
	}
	return fmt.Sprintf("`%s`", f.Location.File)
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
