package sign_test

// Gap tests for sign.go: Sign write error, Verify invalid hex signature, and
// Verify missing-artifact (hmacFile open failure) branches.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SoundMatt/FuSaOps/sign"
)

// TestSignWriteError verifies Sign returns an error when the signature file
// cannot be written, covering sign.go:73.  The artifact path is placed so
// that path+".sig" resolves to an existing directory — os.WriteFile on a
// directory path returns an error on all supported platforms.
//
//fusa:test REQ-FO-SIGN003
func TestSignWriteError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based setup not applicable on Windows")
	}
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("content"), 0o640); err != nil {
		t.Fatal(err)
	}
	// Create artifact.zip.sig as a directory so os.WriteFile fails.
	sigDir := artifact + sign.SigExt
	if err := os.Mkdir(sigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	_, err := sign.Sign(artifact, key)
	if err == nil {
		t.Error("Sign: expected error when sig path is a directory, got nil")
	}
}

// TestVerifyBadSigHex verifies Verify returns an error when the signature
// file contains invalid hex, covering sign.go:94.
//
//fusa:test REQ-FO-SIGN004
func TestVerifyBadSigHex(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("content"), 0o640); err != nil {
		t.Fatal(err)
	}
	// Write a non-hex signature file.
	sigPath := artifact + sign.SigExt
	if err := os.WriteFile(sigPath, []byte("NOTVALIDHEX!!\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	err := sign.Verify(artifact, key)
	if err == nil {
		t.Error("Verify: expected error for non-hex sig, got nil")
	}
}

// TestVerifyMissingArtifactWithSig verifies Verify returns an error when the
// artifact file is missing but the signature file exists, covering the
// hmacFile os.Open failure path at sign.go:98.
//
//fusa:test REQ-FO-SIGN004
func TestVerifyMissingArtifactWithSig(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("content"), 0o640); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	// Sign the artifact first to get a valid .sig.
	if _, err := sign.Sign(artifact, key); err != nil {
		t.Fatalf("Sign setup: %v", err)
	}
	// Remove the artifact — .sig remains.
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}
	err := sign.Verify(artifact, key)
	if err == nil {
		t.Error("Verify: expected error for missing artifact, got nil")
	}
}
