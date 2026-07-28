package fusaops

import (
	"crypto/sha256"
	"fmt"
)

// AttestationStatus records whether an evidence artifact's qualitative
// content has been reviewed by an independent human, per x-FuSa spec §1.6.2.
//
//fusa:req REQ-FO-CORE009
type AttestationStatus string

const (
	// AttestationHeuristic is the fail-safe default: no human has vouched for
	// this content. Absent status MUST be treated as this value.
	AttestationHeuristic AttestationStatus = "heuristic"
	// AttestationReviewed asserts an independent reviewer examined the
	// content at ContentHash and found it genuine, not templated/placeholder.
	AttestationReviewed AttestationStatus = "reviewed"
)

// Attestation is a DCO-style, per-artifact provenance record. It lets a
// consumer distinguish auto-scaffolded content from human-reviewed content
// without requiring a detection heuristic to be perfectly precise: rather
// than proving the content is good, it requires a named, independent human
// to assert that they checked, with an accountability trail attached. The
// independence model (ImplementationAuthor vs IndependentReviewer) mirrors
// the one already used by the vv package (ISO 26262-2:2018 §6.4).
//
//fusa:req REQ-FO-CORE009
type Attestation struct {
	Status               AttestationStatus `json:"status"`
	ImplementationAuthor string            `json:"implementationAuthor,omitempty"`
	IndependentReviewer  string            `json:"independentReviewer,omitempty"`
	ReviewedAt           string            `json:"reviewedAt,omitempty"` // RFC 3339
	ContentHash          string            `json:"contentHash,omitempty"`
}

// AttestationValid reports whether a is a non-stale, genuinely independent
// "reviewed" attestation matching currentContentHash. Every failure mode —
// absent attestation, status "heuristic", a self-attestation (reviewer ==
// author), or a stale hash — returns false (fail-safe): the caller MUST then
// treat the content as unreviewed.
//
//fusa:req REQ-FO-CORE009
func AttestationValid(a Attestation, currentContentHash string) bool {
	if a.Status != AttestationReviewed {
		return false
	}
	if a.IndependentReviewer == "" || a.IndependentReviewer == a.ImplementationAuthor {
		return false
	}
	if a.ContentHash == "" || currentContentHash == "" {
		return false
	}
	return a.ContentHash == currentContentHash
}

// ContentHash computes the sha256:-prefixed RFC 8785 canonical hash of data,
// suitable for populating Attestation.ContentHash or for comparison in
// AttestationValid. Callers pass the JSON-marshaled artifact content with
// the Attestation field itself (and any generatedAt-style timestamp) already
// excluded, so a later edit to that content invalidates a prior attestation.
//
//fusa:req REQ-FO-CORE009
func ContentHash(data []byte) string {
	canon, err := Canonicalize(data)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canon)
	return fmt.Sprintf("sha256:%x", sum)
}
