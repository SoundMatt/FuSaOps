package fusaops

import "testing"

//fusa:test REQ-FO-CORE009
func TestAttestationValidTrue(t *testing.T) {
	hash := ContentHash([]byte(`{"a":1}`))
	a := Attestation{
		Status:               AttestationReviewed,
		ImplementationAuthor: "auto",
		IndependentReviewer:  "Jane Doe <jane@example.com>",
		ReviewedAt:           "2026-07-28T00:00:00Z",
		ContentHash:          hash,
	}
	if !AttestationValid(a, hash) {
		t.Error("expected valid attestation")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidHeuristicStatus(t *testing.T) {
	hash := ContentHash([]byte(`{"a":1}`))
	a := Attestation{Status: AttestationHeuristic, ContentHash: hash}
	if AttestationValid(a, hash) {
		t.Error("heuristic status must never be valid")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidAbsentStatus(t *testing.T) {
	hash := ContentHash([]byte(`{"a":1}`))
	a := Attestation{ContentHash: hash}
	if AttestationValid(a, hash) {
		t.Error("absent status must default to invalid (fail-safe)")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidSelfAttestation(t *testing.T) {
	hash := ContentHash([]byte(`{"a":1}`))
	a := Attestation{
		Status:               AttestationReviewed,
		ImplementationAuthor: "Jane Doe <jane@example.com>",
		IndependentReviewer:  "Jane Doe <jane@example.com>",
		ContentHash:          hash,
	}
	if AttestationValid(a, hash) {
		t.Error("a reviewer identical to the author must not satisfy independence")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidEmptyReviewer(t *testing.T) {
	hash := ContentHash([]byte(`{"a":1}`))
	a := Attestation{Status: AttestationReviewed, ContentHash: hash}
	if AttestationValid(a, hash) {
		t.Error("an empty IndependentReviewer must not satisfy independence")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidStaleHash(t *testing.T) {
	oldHash := ContentHash([]byte(`{"a":1}`))
	newHash := ContentHash([]byte(`{"a":2}`))
	a := Attestation{
		Status:              AttestationReviewed,
		IndependentReviewer: "Jane Doe <jane@example.com>",
		ContentHash:         oldHash,
	}
	if AttestationValid(a, newHash) {
		t.Error("a stale content hash must invalidate the attestation")
	}
}

//fusa:test REQ-FO-CORE009
func TestAttestationValidEmptyHashes(t *testing.T) {
	a := Attestation{Status: AttestationReviewed, IndependentReviewer: "Jane Doe"}
	if AttestationValid(a, "") {
		t.Error("empty content hashes must not be considered valid")
	}
}

//fusa:test REQ-FO-CORE009
func TestContentHashDeterministic(t *testing.T) {
	h1 := ContentHash([]byte(`{"a":1,"b":2}`))
	h2 := ContentHash([]byte(`{"b":2,"a":1}`))
	if h1 != h2 {
		t.Errorf("ContentHash should be key-order independent: %q != %q", h1, h2)
	}
	if h1 == "" {
		t.Error("ContentHash must not be empty for valid JSON")
	}
}

//fusa:test REQ-FO-CORE009
func TestContentHashInvalidJSON(t *testing.T) {
	if got := ContentHash([]byte("not json")); got != "" {
		t.Errorf("ContentHash of invalid JSON should be empty, got %q", got)
	}
}
