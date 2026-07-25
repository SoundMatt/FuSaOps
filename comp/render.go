package comp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Render writes the Aggregate to w in the given format (text, json).
//
//fusa:req REQ-FO-COMP003
func Render(w io.Writer, agg *Aggregate, format string) error {
	switch format {
	case "text", "":
		return renderText(w, agg)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(agg)
	default:
		return fmt.Errorf("comp: unsupported format %q", format)
	}
}

// RenderToFile writes the Aggregate to path in the given format.
//
//fusa:req REQ-FO-COMP003
func RenderToFile(agg *Aggregate, format, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("comp: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Render(f, agg, format)
}

func renderText(w io.Writer, agg *Aggregate) error {
	for _, c := range agg.Components {
		if c.Skipped != "" {
			fmt.Fprintf(w, "[%s/%s] SKIPPED: %s\n", c.Language, c.Tool, c.Skipped)
			continue
		}
		if c.Report == nil {
			continue
		}
		r := c.Report
		status := "PASS"
		if r.Violations > 0 {
			status = "FAIL"
		}
		dal := ""
		if r.DAL != "" {
			dal = " (" + r.DAL + ")"
		}
		fmt.Fprintf(w, "[%s/%s] %s — threshold %d%s, %d functions, %d violations\n",
			c.Language, c.Tool, status, r.Threshold, dal, r.TotalFunctions, r.Violations)
		for _, fn := range r.Results {
			if fn.ExceedsThreshold {
				fmt.Fprintf(w, "  %-40s  %s:%d  V(G)=%d\n", fn.Name, fn.File, fn.Line, fn.Complexity)
			}
		}
	}
	status := "PASS"
	if agg.Violations > 0 {
		status = "FAIL"
	}
	fmt.Fprintf(w, "\nTOTAL %s — %d functions, %d violations\n", status, agg.TotalFunctions, agg.Violations)
	return nil
}
