// Package fleet provides multi-repository safety analysis orchestration.
//
// A Fleet is a list of named repository roots; Run executes fusaops check
// against each root in parallel and aggregates the results into a FleetReport.
// The fleet config is a simple JSON file — see Config for the format.
package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// Config is the JSON configuration for a fleet run.
//
//fusa:req REQ-FO-FLT001
type Config struct {
	Project string `json:"project"`
	Repos   []Repo `json:"repos"`
}

// Repo is a single repository entry in the fleet config.
//
//fusa:req REQ-FO-FLT001
type Repo struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	Adapter string `json:"adapter,omitempty"`
}

// RepoResult holds the outcome of scanning one repository.
//
//fusa:req REQ-FO-FLT002
type RepoResult struct {
	Name     string `json:"name"`
	Dir      string `json:"dir"`
	Status   string `json:"status"`
	Total    int    `json:"total"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Infos    int    `json:"infos"`
	ScanErr  string `json:"error,omitempty"`
}

// FleetReport is the aggregate result across all repos in a fleet.
//
//fusa:req REQ-FO-FLT002
type FleetReport struct {
	Project     string       `json:"project"`
	GeneratedAt time.Time    `json:"generatedAt"`
	Repos       []RepoResult `json:"repos"`
	Total       int          `json:"total"`
	Errors      int          `json:"errors"`
	Warnings    int          `json:"warnings"`
	Infos       int          `json:"infos"`
}

// Status returns the overall fleet verdict: FAIL if any repo has FAIL or ERROR,
// WARN if any has WARN, otherwise PASS.
//
//fusa:req REQ-FO-FLT002
func (r *FleetReport) Status() string {
	for _, repo := range r.Repos {
		if repo.Status == "FAIL" || repo.Status == "ERROR" {
			return "FAIL"
		}
	}
	for _, repo := range r.Repos {
		if repo.Status == "WARN" {
			return "WARN"
		}
	}
	return "PASS"
}

// LoadConfig reads and parses a fleet JSON config file.
//
//fusa:req REQ-FO-FLT001
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("fleet: read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("fleet: parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Run executes fusaops check against each repo in cfg in parallel and returns
// an aggregated FleetReport.
//
//fusa:req REQ-FO-FLT003
func Run(ctx context.Context, cfg Config, runner *orchestrator.Runner) *FleetReport {
	results := make([]RepoResult, len(cfg.Repos))
	var wg sync.WaitGroup
	for i, repo := range cfg.Repos {
		wg.Add(1)
		go func(i int, repo Repo) {
			defer wg.Done()
			opts := orchestrator.Options{Project: repo.Name}
			if repo.Adapter != "" {
				opts.Only = []string{repo.Adapter}
			}
			rep, err := runner.Run(ctx, repo.Dir, opts)
			rr := RepoResult{Name: repo.Name, Dir: repo.Dir}
			if err != nil {
				rr.Status = "ERROR"
				rr.ScanErr = err.Error()
			} else {
				rr.Status = rep.Summary.Status()
				rr.Total = rep.Summary.Total
				rr.Errors = rep.Summary.Errors
				rr.Warnings = rep.Summary.Warnings
				rr.Infos = rep.Summary.Infos
			}
			results[i] = rr
		}(i, repo)
	}
	wg.Wait()

	fr := &FleetReport{
		Project:     cfg.Project,
		GeneratedAt: time.Now().UTC(),
		Repos:       results,
	}
	for _, r := range results {
		fr.Total += r.Total
		fr.Errors += r.Errors
		fr.Warnings += r.Warnings
		fr.Infos += r.Infos
	}
	return fr
}

// Render writes the FleetReport to w in the requested format (text, json, html, or markdown).
//
//fusa:req REQ-FO-FLT004
//fusa:req REQ-FO-FLT005
//fusa:req REQ-FO-FLT007
func Render(w io.Writer, fr *FleetReport, format string) error {
	switch format {
	case "json", "":
		return renderJSON(w, fr)
	case "text":
		return renderText(w, fr)
	case "html":
		return renderHTML(w, fr)
	case "markdown", "md":
		return renderMarkdown(w, fr)
	default:
		return fmt.Errorf("fleet: unsupported format %q (want text, json, html, or markdown)", format)
	}
}

func renderJSON(w io.Writer, fr *FleetReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(fr)
}

func renderText(w io.Writer, fr *FleetReport) error {
	fmt.Fprintf(w, "Fleet: %s  [%s]\n", fr.Project, fr.Status())
	fmt.Fprintf(w, "Generated: %s\n\n", fr.GeneratedAt.UTC().Format("2006-01-02 15:04:05 MST"))
	maxName := 4
	for _, r := range fr.Repos {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}
	fmt.Fprintf(w, "%-*s  %-6s  %5s  %6s  %5s\n", maxName, "REPO", "STATUS", "TOTAL", "ERRORS", "WARNS")
	fmt.Fprintf(w, "%s\n", repeatChar('-', maxName+32))
	for _, r := range fr.Repos {
		if r.ScanErr != "" {
			fmt.Fprintf(w, "%-*s  %-6s  (scan error: %s)\n", maxName, r.Name, "ERROR", r.ScanErr)
		} else {
			fmt.Fprintf(w, "%-*s  %-6s  %5d  %6d  %5d\n", maxName, r.Name, r.Status, r.Total, r.Errors, r.Warnings)
		}
	}
	fmt.Fprintf(w, "%s\n", repeatChar('-', maxName+32))
	fmt.Fprintf(w, "%-*s  %-6s  %5d  %6d  %5d\n", maxName, "TOTAL", fr.Status(), fr.Total, fr.Errors, fr.Warnings)
	return nil
}

var fleetHTMLTmpl = template.Must(template.New("fleet").Funcs(template.FuncMap{
	"badgeClass": func(status string) string {
		switch status {
		case "PASS":
			return "badge-pass"
		case "WARN":
			return "badge-warn"
		case "FAIL", "ERROR":
			return "badge-fail"
		default:
			return "badge-skip"
		}
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps Fleet — {{.Project}}</title>
<style>
:root{--pass:#22c55e;--warn:#f59e0b;--fail:#ef4444;--skip:#94a3b8;--bg:#f8fafc;--card:#fff;--muted:#64748b}
body{font-family:system-ui,sans-serif;background:var(--bg);margin:0;padding:1.5rem}
h1{margin:0 0 .25rem;font-size:1.4rem}
.meta{color:var(--muted);font-size:.85rem;margin-bottom:1.5rem}
.badge{display:inline-block;padding:.2rem .6rem;border-radius:4px;font-weight:700;font-size:.8rem;color:#fff}
.badge-pass{background:var(--pass)}.badge-warn{background:var(--warn)}.badge-fail{background:var(--fail)}.badge-skip{background:var(--skip)}
table{border-collapse:collapse;width:100%;background:var(--card);border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,.08)}
th{background:#f1f5f9;text-align:left;padding:.6rem 1rem;font-size:.8rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted)}
td{padding:.6rem 1rem;border-top:1px solid #e2e8f0;font-size:.9rem}
tr:hover td{background:#f8fafc}
.num{text-align:right;font-variant-numeric:tabular-nums}
.err{color:var(--fail);font-weight:600}
.warn{color:var(--warn);font-weight:600}
.scan-err{color:var(--fail);font-size:.85rem}
tfoot td{font-weight:600;border-top:2px solid #e2e8f0;background:#f8fafc}
</style>
</head>
<body>
<h1>FuSaOps Fleet — {{.Project}}</h1>
<p class="meta">Generated {{.GeneratedAt.UTC.Format "2006-01-02 15:04:05 UTC"}} &nbsp;|&nbsp; <span class="badge {{badgeClass .Status}}">{{.Status}}</span></p>
<table>
<thead><tr>
<th>Repository</th><th>Status</th><th class="num">Errors</th><th class="num">Warnings</th><th class="num">Infos</th><th class="num">Total</th>
</tr></thead>
<tbody>
{{range .Repos}}<tr>
<td>{{.Name}}</td>
<td><span class="badge {{badgeClass .Status}}">{{.Status}}</span></td>
{{if .ScanErr}}
<td colspan="4" class="scan-err">⚠ {{.ScanErr}}</td>
{{else}}
<td class="num{{if .Errors}} err{{end}}">{{.Errors}}</td>
<td class="num{{if .Warnings}} warn{{end}}">{{.Warnings}}</td>
<td class="num">{{.Infos}}</td>
<td class="num">{{.Total}}</td>
{{end}}
</tr>{{end}}
</tbody>
<tfoot><tr>
<td>TOTAL</td>
<td><span class="badge {{badgeClass .Status}}">{{.Status}}</span></td>
<td class="num{{if .Errors}} err{{end}}">{{.Errors}}</td>
<td class="num{{if .Warnings}} warn{{end}}">{{.Warnings}}</td>
<td class="num">{{.Infos}}</td>
<td class="num">{{.Total}}</td>
</tr></tfoot>
</table>
</body>
</html>
`))

// renderHTML writes a self-contained HTML fleet report to w.
//
//fusa:req REQ-FO-FLT005
func renderHTML(w io.Writer, fr *FleetReport) error {
	return fleetHTMLTmpl.Execute(w, fr)
}

// renderMarkdown writes a GFM markdown fleet report to w.
//
//fusa:req REQ-FO-FLT007
func renderMarkdown(w io.Writer, fr *FleetReport) error {
	status := fr.Status()
	badge := "🟢"
	switch status {
	case "WARN":
		badge = "🟡"
	case "FAIL", "ERROR":
		badge = "🔴"
	}
	fmt.Fprintf(w, "# FuSaOps Fleet — %s\n\n", fr.Project)
	fmt.Fprintf(w, "%s **%s** · Generated %s\n\n", badge, status, fr.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))
	fmt.Fprintln(w, "| Repository | Status | Errors | Warnings | Infos | Total |")
	fmt.Fprintln(w, "|---|---|---:|---:|---:|---:|")
	for _, r := range fr.Repos {
		if r.ScanErr != "" {
			fmt.Fprintf(w, "| %s | **%s** | ⚠ %s | | | |\n", r.Name, r.Status, r.ScanErr)
		} else {
			fmt.Fprintf(w, "| %s | **%s** | %d | %d | %d | %d |\n",
				r.Name, r.Status, r.Errors, r.Warnings, r.Infos, r.Total)
		}
	}
	fmt.Fprintf(w, "| **TOTAL** | **%s** | **%d** | **%d** | **%d** | **%d** |\n",
		status, fr.Errors, fr.Warnings, fr.Infos, fr.Total)
	return nil
}

// RenderToFile writes the fleet report to path, or to w if path is empty.
//
//fusa:req REQ-FO-FLT004
func RenderToFile(w io.Writer, fr *FleetReport, format, path string) error {
	if path == "" {
		return Render(w, fr, format)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("fleet: create output file: %w", err)
	}
	defer f.Close()
	return Render(f, fr, format)
}

func repeatChar(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

// HasFailures returns true when the fleet has at least one FAIL or ERROR repo.
//
//fusa:req REQ-FO-FLT002
func (r *FleetReport) HasFailures() bool {
	return r.Status() == "FAIL"
}

// HasWarnings returns true when the fleet has warnings but no failures.
//
//fusa:req REQ-FO-FLT002
func (r *FleetReport) HasWarnings() bool {
	return r.Status() == "WARN"
}
