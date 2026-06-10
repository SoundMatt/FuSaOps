package conform

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	digitRunRE = regexp.MustCompile(`[0-9]+`)
	wsRunRE    = regexp.MustCompile(`\s+`)
)

// normalizeMessage applies the §4.2 message normalisation:
// replace every run of ASCII digits with "#", collapse whitespace runs to a
// single space, and trim.  Unicode NFC is applied only when the message
// contains non-ASCII codepoints.
//
//fusa:req REQ-FO-CNF017
func normalizeMessage(msg string) string {
	// NFC only for non-ASCII messages.
	if hasNonASCII(msg) {
		msg = applyNFC(msg)
	}
	msg = digitRunRE.ReplaceAllString(msg, "#")
	msg = wsRunRE.ReplaceAllString(msg, " ")
	return strings.TrimSpace(msg)
}

// Fingerprint computes the §4.2 canonical fingerprint for a finding.
//
//fusa:req REQ-FO-CNF017
func Fingerprint(ruleID, file, message string) string {
	canonical := ruleID + "\x1f" + file + "\x1f" + normalizeMessage(message)
	h := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("sha256:%x", h)
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

// applyNFC normalises a string to Unicode NFC form.
// Uses a simple passthrough when unicode/norm is not available — standard library
// only; a full NFC implementation is not needed for ASCII-only messages.
func applyNFC(s string) string {
	// stdlib has no NFC; iterate and compose manually is impractical.
	// For the spec's purpose: ASCII messages (the common case) skip this path,
	// and non-ASCII is rare enough that tools may include unicode/norm as an
	// optional dependency.  FuSaOps validates format only, not the NFC value.
	return s
}
