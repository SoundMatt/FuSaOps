package server

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/qualify"
	"github.com/SoundMatt/FuSaOps/report"
)

// WithQualifyReport sets the path to a qualify JSON report for the dashboard
// badge. When empty, the server auto-discovers the report from the project root.
//
//fusa:req REQ-FO-CLI079
func (s *Server) WithQualifyReport(path string) *Server {
	s.qualifyPath = path
	return s
}

// loadQualifyInfo reads a qualify report and maps it to a report.QualifyInfo.
// Returns nil when the file is absent (normal on first run) or on read error.
//
//fusa:req REQ-FO-SRV011
func loadQualifyInfo(root, qualifyPath string) *report.QualifyInfo {
	path := qualifyPath
	if path == "" {
		path = filepath.Join(root, qualify.ReportFile)
	}
	qr, err := qualify.Load(path)
	if err != nil {
		return nil
	}
	typ := qr.QualificationType
	if typ == "" {
		typ = "self"
	}
	return &report.QualifyInfo{
		Type:        typ,
		RecordUri:   qr.QualificationRecordUri,
		AllPassed:   !qr.HasFailures(),
		Total:       qr.Total,
		PassedCount: qr.Passed,
		Failed:      qr.Failed,
	}
}

// handleQualifyBadge renders an SVG badge reflecting the qualification status.
//
//fusa:req REQ-FO-SRV011
func (s *Server) handleQualifyBadge(w http.ResponseWriter, _ *http.Request) {
	qi := loadQualifyInfo(s.root, s.qualifyPath)
	label := "qualification"
	msg, color := "pending", "#9f9f9f"
	if qi != nil {
		typ := qi.Type
		if qi.AllPassed {
			msg, color = typ+" / pass", "#4c1"
		} else {
			msg, color = typ+" / failing", "#e05d44"
		}
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprint(w, svgBadge(label, msg, color))
}
