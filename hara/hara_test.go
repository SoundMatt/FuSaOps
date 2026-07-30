package hara_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/hara"
)

//fusa:test REQ-FO-HARA001
func TestConstants(t *testing.T) {
	if hara.HARAFile == "" {
		t.Fatal("HARAFile must not be empty")
	}
	if !strings.HasSuffix(hara.HARAFile, ".json") {
		t.Errorf("HARAFile should end in .json, got %q", hara.HARAFile)
	}
}

//fusa:test REQ-FO-HARA001
func TestASILConstants(t *testing.T) {
	for _, a := range []hara.ASIL{hara.ASILQM, hara.ASILA, hara.ASILB, hara.ASILC, hara.ASILD} {
		if string(a) == "" {
			t.Error("ASIL constant must not be empty")
		}
	}
	for _, s := range []hara.Severity{hara.SeverityS0, hara.SeverityS1, hara.SeverityS2, hara.SeverityS3} {
		if string(s) == "" {
			t.Error("Severity constant must not be empty")
		}
	}
	for _, e := range []hara.Exposure{hara.ExposureE0, hara.ExposureE1, hara.ExposureE2, hara.ExposureE3, hara.ExposureE4} {
		if string(e) == "" {
			t.Error("Exposure constant must not be empty")
		}
	}
	for _, c := range []hara.Controllability{hara.ControllabilityC0, hara.ControllabilityC1, hara.ControllabilityC2, hara.ControllabilityC3} {
		if string(c) == "" {
			t.Error("Controllability constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-HARA004
func TestDetermineASILS0AlwaysQM(t *testing.T) {
	for _, e := range []hara.Exposure{hara.ExposureE1, hara.ExposureE2, hara.ExposureE3, hara.ExposureE4} {
		for _, c := range []hara.Controllability{hara.ControllabilityC0, hara.ControllabilityC1, hara.ControllabilityC2, hara.ControllabilityC3} {
			got := hara.DetermineASIL(hara.SeverityS0, e, c)
			if got != hara.ASILQM {
				t.Errorf("S0 should always be QM, got %s (E=%s C=%s)", got, e, c)
			}
		}
	}
}

//fusa:test REQ-FO-HARA004
func TestDetermineASILE0AlwaysQM(t *testing.T) {
	for _, s := range []hara.Severity{hara.SeverityS1, hara.SeverityS2, hara.SeverityS3} {
		for _, c := range []hara.Controllability{hara.ControllabilityC0, hara.ControllabilityC1, hara.ControllabilityC2, hara.ControllabilityC3} {
			got := hara.DetermineASIL(s, hara.ExposureE0, c)
			if got != hara.ASILQM {
				t.Errorf("E0 should always be QM, got %s (S=%s C=%s)", got, s, c)
			}
		}
	}
}

//fusa:test REQ-FO-HARA004
func TestDetermineASILKnownValues(t *testing.T) {
	// Corrected against ISO 26262-3:2018 Table 4. C0 (controllable in general)
	// maps to QM; C1..C3 follow the canonical table.
	cases := []struct {
		s    hara.Severity
		e    hara.Exposure
		c    hara.Controllability
		want hara.ASIL
	}{
		{hara.SeverityS2, hara.ExposureE3, hara.ControllabilityC2, hara.ASILA},
		{hara.SeverityS3, hara.ExposureE4, hara.ControllabilityC3, hara.ASILD},
		{hara.SeverityS1, hara.ExposureE1, hara.ControllabilityC0, hara.ASILQM},
		{hara.SeverityS3, hara.ExposureE1, hara.ControllabilityC3, hara.ASILA},
		{hara.SeverityS2, hara.ExposureE4, hara.ControllabilityC0, hara.ASILQM},
	}
	for _, tc := range cases {
		got := hara.DetermineASIL(tc.s, tc.e, tc.c)
		if got != tc.want {
			t.Errorf("DetermineASIL(%s,%s,%s)=%s, want %s", tc.s, tc.e, tc.c, got, tc.want)
		}
	}
}

//fusa:test REQ-FO-HARA004
func TestDetermineASILAll36Cells(t *testing.T) {
	// Independent oracle for ISO 26262-3:2018 Table 4: for C1..C3 the ASIL is a
	// function of the additive index S+E+C (S1=1..S3=3, E1=1..E4=4, C1=1..C3=3):
	// <=6 QM, 7 A, 8 B, 9 C, 10 D. C0 is always QM.
	sVals := []struct {
		s hara.Severity
		n int
	}{{hara.SeverityS1, 1}, {hara.SeverityS2, 2}, {hara.SeverityS3, 3}}
	eVals := []struct {
		e hara.Exposure
		n int
	}{{hara.ExposureE1, 1}, {hara.ExposureE2, 2}, {hara.ExposureE3, 3}, {hara.ExposureE4, 4}}
	cVals := []struct {
		c hara.Controllability
		n int
	}{{hara.ControllabilityC1, 1}, {hara.ControllabilityC2, 2}, {hara.ControllabilityC3, 3}}
	oracle := func(t int) hara.ASIL {
		switch {
		case t >= 10:
			return hara.ASILD
		case t == 9:
			return hara.ASILC
		case t == 8:
			return hara.ASILB
		case t == 7:
			return hara.ASILA
		default:
			return hara.ASILQM
		}
	}
	for _, sv := range sVals {
		for _, ev := range eVals {
			for _, cv := range cVals {
				want := oracle(sv.n + ev.n + cv.n)
				if got := hara.DetermineASIL(sv.s, ev.e, cv.c); got != want {
					t.Errorf("DetermineASIL(%s,%s,%s)=%s, want %s", sv.s, ev.e, cv.c, got, want)
				}
			}
			// C0 always QM.
			if got := hara.DetermineASIL(sv.s, ev.e, hara.ControllabilityC0); got != hara.ASILQM {
				t.Errorf("DetermineASIL(%s,%s,C0)=%s, want QM", sv.s, ev.e, got)
			}
		}
	}
}

//fusa:test REQ-FO-HARA002
func TestMaxASIL(t *testing.T) {
	hazards := []hara.Hazard{
		{ID: "H-001", Risk: hara.RiskRating{ASIL: hara.ASILB}},
		{ID: "H-002", Risk: hara.RiskRating{ASIL: hara.ASILD}},
		{ID: "H-003", Risk: hara.RiskRating{ASIL: hara.ASILA}},
	}
	got := hara.MaxASIL(hazards)
	if got != hara.ASILD {
		t.Errorf("MaxASIL=%s, want ASIL-D", got)
	}
}

//fusa:test REQ-FO-HARA002
func TestMaxASILDerived(t *testing.T) {
	hazards := []hara.Hazard{
		{ID: "H-001", Risk: hara.RiskRating{
			Severity: hara.SeverityS3, Exposure: hara.ExposureE4, Controllability: hara.ControllabilityC3,
		}},
	}
	got := hara.MaxASIL(hazards)
	if got != hara.ASILD {
		t.Errorf("MaxASIL derived=%s, want ASIL-D", got)
	}
}

//fusa:test REQ-FO-HARA002
func TestValidateComplete(t *testing.T) {
	h := &hara.HARA{
		Hazards: []hara.Hazard{
			{
				ID: "H-001", Description: "test hazard",
				Risk:        hara.RiskRating{Severity: hara.SeverityS2, Exposure: hara.ExposureE3, Controllability: hara.ControllabilityC2, ASIL: hara.ASILA},
				SafetyGoals: []string{"SG-001"},
			},
		},
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", Description: "test goal", ASIL: hara.ASILA, FssrRefs: []string{"REQ-FO-HARA001"}},
		},
	}
	findings := hara.Validate(h)
	if len(findings) != 0 {
		t.Errorf("expected no findings for complete HARA, got %d: %v", len(findings), findings)
	}
}

//fusa:test REQ-FO-HARA002
func TestValidateIncompleteRating(t *testing.T) {
	h := &hara.HARA{
		Hazards: []hara.Hazard{
			{ID: "H-001", Risk: hara.RiskRating{Severity: hara.SeverityS2}, SafetyGoals: []string{"SG-001"}},
		},
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", ASIL: hara.ASILB},
		},
	}
	findings := hara.Validate(h)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Message, "incomplete risk rating") {
			found = true
		}
	}
	if !found {
		t.Error("expected incomplete risk rating finding")
	}
}

//fusa:test REQ-FO-HARA002
func TestValidateNoSafetyGoal(t *testing.T) {
	h := &hara.HARA{
		Hazards: []hara.Hazard{
			{
				ID: "H-001",
				Risk: hara.RiskRating{
					Severity: hara.SeverityS2, Exposure: hara.ExposureE3, Controllability: hara.ControllabilityC2,
				},
			},
		},
	}
	findings := hara.Validate(h)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Message, "no linked safety goal") {
			found = true
		}
	}
	if !found {
		t.Error("expected no-safety-goal finding")
	}
}

//fusa:test REQ-FO-HARA002
func TestValidateUnknownGoalRef(t *testing.T) {
	h := &hara.HARA{
		Hazards: []hara.Hazard{
			{
				ID: "H-001",
				Risk: hara.RiskRating{
					Severity: hara.SeverityS2, Exposure: hara.ExposureE3,
					Controllability: hara.ControllabilityC2, ASIL: hara.ASILB,
				},
				SafetyGoals: []string{"SG-MISSING"},
			},
		},
	}
	findings := hara.Validate(h)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Message, "unknown safety goal") {
			found = true
		}
	}
	if !found {
		t.Error("expected unknown-safety-goal finding")
	}
}

//fusa:test REQ-FO-HARA002
func TestValidateNoASILOnGoal(t *testing.T) {
	h := &hara.HARA{
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", Description: "no asil"},
		},
	}
	findings := hara.Validate(h)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Message, "no ASIL assigned") {
			found = true
		}
	}
	if !found {
		t.Error("expected no-ASIL finding on safety goal")
	}
}

//fusa:test REQ-FO-HARA005
func TestValidateNoFssrRefsOnGoal(t *testing.T) {
	h := &hara.HARA{
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", Description: "no fssrRefs", ASIL: hara.ASILB},
		},
	}
	findings := hara.Validate(h)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Message, "no fssrRefs") {
			found = true
		}
	}
	if !found {
		t.Error("expected no-fssrRefs finding on safety goal")
	}
}

// TestAttestationContentHashStableAndExcludesAttestation verifies
// AttestationContentHash is deterministic for identical content and does
// not change when only the Attestation field itself is set (it must be
// excluded from its own input, or a chicken-and-egg staleness loop would
// result).
//
//fusa:test REQ-FO-HARA006
func TestAttestationContentHashStableAndExcludesAttestation(t *testing.T) {
	h := &hara.HARA{Project: "test", Standard: "ISO 26262"}
	h1 := hara.AttestationContentHash(h)
	h2 := hara.AttestationContentHash(h)
	if h1 == "" {
		t.Error("AttestationContentHash must not be empty")
	}
	if h1 != h2 {
		t.Errorf("AttestationContentHash not deterministic: %q != %q", h1, h2)
	}

	h.Attestation = &fusaops.Attestation{Status: fusaops.AttestationReviewed, ContentHash: h1}
	h3 := hara.AttestationContentHash(h)
	if h3 != h1 {
		t.Errorf("AttestationContentHash changed after setting Attestation: %q != %q", h3, h1)
	}
}

//fusa:test REQ-FO-HARA003
func TestLoadMissing(t *testing.T) {
	h, err := hara.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load of missing file should return empty HARA, got error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil HARA")
	}
	if len(h.Hazards) != 0 {
		t.Errorf("empty HARA should have no hazards, got %d", len(h.Hazards))
	}
}

//fusa:test REQ-FO-HARA003
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, hara.HARAFile), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := hara.Load(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

//fusa:test REQ-FO-HARA003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	h := &hara.HARA{
		Project:   "TestProject",
		Standard:  "ISO 26262",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		Hazards: []hara.Hazard{
			{
				ID: "H-001", Description: "test",
				Risk:        hara.RiskRating{Severity: hara.SeverityS2, Exposure: hara.ExposureE3, Controllability: hara.ControllabilityC2, ASIL: hara.ASILB},
				SafetyGoals: []string{"SG-001"},
			},
		},
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", Description: "goal", ASIL: hara.ASILB, SafeState: "safe"},
		},
	}
	path := filepath.Join(dir, hara.HARAFile)
	if err := hara.Save(path, h); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := hara.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Project != h.Project {
		t.Errorf("Project=%q, want %q", loaded.Project, h.Project)
	}
	if len(loaded.Hazards) != 1 {
		t.Errorf("Hazards=%d, want 1", len(loaded.Hazards))
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderText(t *testing.T) {
	h := &hara.HARA{
		Project:  "Proj",
		Standard: "ISO 26262",
		Hazards: []hara.Hazard{
			{
				ID: "H-001", Description: "desc",
				Risk:        hara.RiskRating{Severity: hara.SeverityS2, Exposure: hara.ExposureE3, Controllability: hara.ControllabilityC2, ASIL: hara.ASILB},
				SafetyGoals: []string{"SG-001"},
			},
		},
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-001", Description: "goal", ASIL: hara.ASILB},
		},
	}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Hazard Analysis") {
		t.Error("text output should contain 'Hazard Analysis'")
	}
	if !strings.Contains(out, "H-001") {
		t.Error("text output should contain hazard ID")
	}
	if !strings.Contains(out, "SG-001") {
		t.Error("text output should contain safety goal ID")
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderJSON(t *testing.T) {
	h := &hara.HARA{Project: "Proj", Standard: "ISO 26262"}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got hara.HARA
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.Project != "Proj" {
		t.Errorf("Project=%q, want %q", got.Project, "Proj")
	}
}

//fusa:test REQ-FO-HARA008
func TestRenderJSONCarriesCommonHeader(t *testing.T) {
	h := &hara.HARA{Project: "Proj", Standard: "iso26262"}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got hara.Report
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.Kind != "hara-report" {
		t.Errorf("Kind = %q, want hara-report", got.Kind)
	}
	if got.SchemaVersion == "" || got.Tool == "" || got.ToolVersion == "" || got.Language == "" {
		t.Errorf("common header fields incomplete: %+v", got)
	}
	if got.Project != "Proj" {
		t.Errorf("embedded HARA.Project = %q, want Proj", got.Project)
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderDefault(t *testing.T) {
	h := &hara.HARA{Project: "Proj"}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("default render must produce output")
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderMarkdown(t *testing.T) {
	h := &hara.HARA{Project: "Proj", Standard: "ISO 26262"}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	if !strings.Contains(buf.String(), "Hazard Analysis") {
		t.Error("markdown output should contain 'Hazard Analysis'")
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderUnknownFormat(t *testing.T) {
	h := &hara.HARA{}
	err := hara.Render(os.Stdout, h, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

//fusa:test REQ-FO-HARA003
func TestRenderGapsSection(t *testing.T) {
	h := &hara.HARA{
		Hazards: []hara.Hazard{
			{ID: "H-001", Risk: hara.RiskRating{Severity: hara.SeverityS2}},
		},
	}
	var buf bytes.Buffer
	if err := hara.Render(&buf, h, "text"); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "Gaps") {
		t.Error("text output should contain Gaps section when validation finds issues")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-HARA003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := hara.Save(path, &hara.HARA{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}

// TestMaxASILUnknownASIL verifies MaxASIL falls back to ASILQM when a hazard
// has an unrecognised ASIL string, exercising the asilRank default return.
//
//fusa:test REQ-FO-HARA002
func TestMaxASILUnknownASIL(t *testing.T) {
	hazards := []hara.Hazard{
		{Risk: hara.RiskRating{ASIL: hara.ASIL("UNRECOGNISED")}},
	}
	got := hara.MaxASIL(hazards)
	if got != hara.ASILQM {
		t.Errorf("MaxASIL with unknown ASIL: got %q, want %q", got, hara.ASILQM)
	}
}

//fusa:test REQ-FO-HARA007
func TestValidateFssrRefs(t *testing.T) {
	h := &hara.HARA{
		SafetyGoals: []hara.SafetyGoal{
			{ID: "SG-01", FssrRefs: []string{"REQ-1", "REQ-2"}},
			{ID: "SG-02", FssrRefs: []string{"REQ-MISSING"}},
		},
	}
	reqIDs := map[string]bool{"REQ-1": true, "REQ-2": true}
	got := hara.ValidateFssrRefs(h, reqIDs)
	if len(got) != 1 {
		t.Fatalf("expected 1 dangling-ref finding, got %d: %+v", len(got), got)
	}
	if got[0].SafetyGoalID != "SG-02" {
		t.Errorf("dangling ref should be on SG-02, got %q", got[0].SafetyGoalID)
	}
	// All-valid case yields no findings.
	h2 := &hara.HARA{SafetyGoals: []hara.SafetyGoal{{ID: "SG-03", FssrRefs: []string{"REQ-1"}}}}
	if n := len(hara.ValidateFssrRefs(h2, reqIDs)); n != 0 {
		t.Errorf("expected no findings for valid refs, got %d", n)
	}
}
