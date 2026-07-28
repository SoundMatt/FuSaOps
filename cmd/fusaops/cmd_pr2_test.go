package main

// Tests covering the x-FuSa spec §1.6.2 attestation carry-forward and
// §1.6.1 quality-gate wiring added to the fmea/tara/hara/safety-case/sas
// commands.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/fmea"
	"github.com/SoundMatt/FuSaOps/hara"
)

// TestFMEAAttestationCarriesForwardWhenValid verifies a hand-added, correctly
// hashed attestation survives a re-run of `fusaops fmea` (which otherwise
// always rebuilds the document from scratch) and is reported valid.
//
//fusa:test REQ-FO-CLI084
func TestFMEAAttestationCarriesForwardWhenValid(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "fmea.json")
	var stdout, stderr bytes.Buffer

	if code := runFMEA([]string{"--dir", dir, "--format", "json", "--output", outPath}, &stdout, &stderr); code > 1 {
		t.Fatalf("first build: unexpected exit %d, stderr: %s", code, stderr.String())
	}

	f, err := fmea.Load(outPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	hash := fmea.AttestationContentHash(f)
	f.Attestation = &fusaops.Attestation{
		Status:               fusaops.AttestationReviewed,
		ImplementationAuthor: "auto",
		IndependentReviewer:  "Jane Doe <jane@example.com>",
		ReviewedAt:           time.Now().UTC().Format(time.RFC3339),
		ContentHash:          hash,
	}
	if saveErr := fmea.Save(outPath, f); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runFMEA([]string{"--dir", dir, "--format", "json", "--output", outPath}, &stdout, &stderr); code > 1 {
		t.Fatalf("second build: unexpected exit %d, stderr: %s", code, stderr.String())
	}

	reloaded, err := fmea.Load(outPath)
	if err != nil {
		t.Fatalf("Load after second build: %v", err)
	}
	if reloaded.Attestation == nil {
		t.Fatal("attestation was lost across a rebuild")
	}
	if !fusaops.AttestationValid(*reloaded.Attestation, fmea.AttestationContentHash(reloaded)) {
		t.Error("carried-forward attestation should still be valid (content unchanged)")
	}
}

// TestFMEANoAttestationOmittedFromJSON verifies a fresh build (no prior file)
// omits the attestation field entirely rather than emitting a zero-value
// object — FuSaOps' own artifacts are never fabricated as "reviewed".
//
//fusa:test REQ-FO-CLI084
func TestFMEANoAttestationOmittedFromJSON(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "fmea.json")
	var stdout, stderr bytes.Buffer
	if code := runFMEA([]string{"--dir", dir, "--format", "json", "--output", outPath}, &stdout, &stderr); code > 1 {
		t.Fatalf("unexpected exit %d, stderr: %s", code, stderr.String())
	}

	var raw map[string]interface{}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, present := raw["attestation"]; present {
		t.Error("attestation key should be omitted (omitempty) when no attestation exists")
	}
}

// TestFMEARequireAttestationFlagParses verifies --require-attestation and
// --strict are accepted flags (wiring smoke test — the 8 hardcoded entries
// are below the >=10 threshold for FUSA-STUB002 to trigger either way).
//
//fusa:test REQ-FO-CLI084
func TestFMEARequireAttestationFlagParses(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runFMEA([]string{"--dir", dir, "--require-attestation", "--format", "json"}, &stdout, &stderr)
	if code > 1 {
		t.Errorf("unexpected exit %d with --require-attestation, stderr: %s", code, stderr.String())
	}
}

// TestHaraAttestationCarriesForwardWhenValid mirrors the FMEA case for the
// Load/Save-based hara command (no Build/regeneration step).
//
//fusa:test REQ-FO-CLI084
func TestHaraAttestationCarriesForwardWhenValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, hara.HARAFile)
	h := &hara.HARA{Project: "test", Standard: "ISO 26262", CreatedAt: time.Now().UTC()}
	if err := hara.Save(path, h); err != nil {
		t.Fatal(err)
	}

	loaded, err := hara.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash := hara.AttestationContentHash(loaded)
	loaded.Attestation = &fusaops.Attestation{
		Status:               fusaops.AttestationReviewed,
		ImplementationAuthor: "auto",
		IndependentReviewer:  "Jane Doe <jane@example.com>",
		ReviewedAt:           time.Now().UTC().Format(time.RFC3339),
		ContentHash:          hash,
	}
	if err := hara.Save(path, loaded); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := runHara([]string{"--dir", dir, "show", "--format", "json"}, &stdout, &stderr); code > 1 {
		t.Fatalf("unexpected exit %d, stderr: %s", code, stderr.String())
	}

	var got hara.HARA
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal show output: %v", err)
	}
	if got.Attestation == nil {
		t.Fatal("attestation missing from show output")
	}
	if !fusaops.AttestationValid(*got.Attestation, hara.AttestationContentHash(&got)) {
		t.Error("attestation should be valid (content unchanged)")
	}
}
