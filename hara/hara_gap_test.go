package hara_test

// Gap tests for hara.go covering the Load read error (line 245), the
// DetermineASIL fallback for an unknown combination (line 197), MaxASIL with an
// explicit ASIL C hazard (covering asilRank ASILC branch, line 209-210), and
// renderText with non-empty Situations and a Hazard carrying multiple SafetyGoals
// (lines 332-334, 346-348).

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/hara"
)

// TestLoadReadError verifies Load returns a non-nil error when the HARA file
// exists but cannot be read (e.g. it is a directory), covering hara.go:245.
//
//fusa:test REQ-FO-HARA003
func TestLoadReadError(t *testing.T) {
	dir := t.TempDir()
	// Create the HARA filename as a directory to trigger a read error.
	if err := os.Mkdir(filepath.Join(dir, hara.HARAFile), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := hara.Load(dir)
	if err == nil {
		t.Fatal("Load: expected error when HARA path is a directory, got nil")
	}
}

// TestDetermineASILUnknownCombo verifies DetermineASIL returns ASILQM when the
// (S, E, C) combination is not present in the lookup table, covering hara.go:197.
//
//fusa:test REQ-FO-HARA002
func TestDetermineASILUnknownCombo(t *testing.T) {
	// "S5" is not a standard severity — not in the table and not S0 — so the
	// fallback return ASILQM at line 197 is reached.
	got := hara.DetermineASIL("S5", hara.ExposureE4, hara.ControllabilityC3)
	if got != hara.ASILQM {
		t.Errorf("DetermineASIL unknown combo: want ASILQM, got %s", got)
	}
}

// TestMaxASILWithC verifies MaxASIL selects ASIL C when a hazard has an
// explicit ASIL C, exercising asilRank(ASILC) == 3 (hara.go:209-210).
//
//fusa:test REQ-FO-HARA002
func TestMaxASILWithC(t *testing.T) {
	hazards := []hara.Hazard{
		{ID: "H-001", Risk: hara.RiskRating{ASIL: hara.ASILC}},
	}
	got := hara.MaxASIL(hazards)
	if got != hara.ASILC {
		t.Errorf("MaxASIL with ASIL-C hazard: want ASILC, got %s", got)
	}
}

// TestRenderTextWithSituationsAndHazards verifies renderText iterates over
// Situations (hara.go:332-334) and produces the ", "-joined SafetyGoals list
// for multi-goal hazards (hara.go:346-348).
//
//fusa:test REQ-FO-HARA004
func TestRenderTextWithSituationsAndHazards(t *testing.T) {
	h := &hara.HARA{
		Project:  "TestProj",
		Standard: "ISO 26262",
		Situations: []hara.OperationalSituation{
			{ID: "OS-001", Description: "Normal highway driving"},
		},
		Hazards: []hara.Hazard{
			{
				ID:          "H-001",
				Description: "Loss of control",
				Risk: hara.RiskRating{
					Severity:        hara.SeverityS3,
					Exposure:        hara.ExposureE4,
					Controllability: hara.ControllabilityC3,
					ASIL:            hara.ASILD,
				},
				// Two SafetyGoals triggers the `if i > 0 { goals += ", " }` branch.
				SafetyGoals: []string{"SG-001", "SG-002"},
			},
		},
	}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "OS-001") {
		t.Errorf("renderText: expected OS-001 situation ID in output:\n%s", out)
	}
	if !strings.Contains(out, "SG-001, SG-002") {
		t.Errorf("renderText: expected 'SG-001, SG-002' in output:\n%s", out)
	}
}
