package report

import (
	"encoding/json"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// SARIF 2.1.0 minimal model — one run per component so each tool's results are
// attributable to the language toolchain that produced them.

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
}

// sarifLevel maps a FuSaOps severity to a SARIF result level.
func sarifLevel(s fusaops.Severity) string {
	switch s {
	case fusaops.SeverityError:
		return "error"
	case fusaops.SeverityWarning:
		return "warning"
	default:
		return "note"
	}
}

// renderSARIF writes the aggregate report as SARIF 2.1.0 for GitHub Code
// Scanning and other SARIF consumers.
//
//fusa:req REQ-FO-RPT011
func renderSARIF(w io.Writer, r *AggregateReport) error {
	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
	}
	for _, c := range r.Components {
		run := sarifRun{
			Tool: sarifTool{Driver: sarifDriver{Name: c.Tool, Version: fusaops.Version}},
		}
		for _, f := range c.Findings {
			res := sarifResult{
				RuleID:  f.RuleID,
				Level:   sarifLevel(f.Severity),
				Message: sarifMessage{Text: f.Message},
			}
			if f.Location.File != "" {
				loc := sarifLocation{PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: f.Location.File},
				}}
				if f.Location.Line > 0 {
					loc.PhysicalLocation.Region = &sarifRegion{
						StartLine:   f.Location.Line,
						StartColumn: f.Location.Column,
					}
				}
				res.Locations = append(res.Locations, loc)
			}
			run.Results = append(run.Results, res)
		}
		log.Runs = append(log.Runs, run)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(log); err != nil {
		return fmt.Errorf("report: sarif encode: %w", err)
	}
	return nil
}
