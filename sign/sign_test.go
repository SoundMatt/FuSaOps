package sign_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/sign"
)

//fusa:test REQ-FO-SIGN001
func TestKeygen(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")

	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// Should be 64 hex chars + newline.
	line := strings.TrimRight(string(data), "\n")
	if len(line) != 64 {
		t.Errorf("key length = %d hex chars, want 64", len(line))
	}
	if _, hexErr := hex.DecodeString(line); hexErr != nil {
		t.Errorf("key is not valid hex: %v", hexErr)
	}

	// File mode should be 0600 (owner read/write only).
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode = %o, want 0600", info.Mode().Perm())
	}
}

//fusa:test REQ-FO-SIGN002
func TestLoadKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	key, err := sign.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d bytes, want 32", len(key))
	}
}

//fusa:test REQ-FO-SIGN002
func TestLoadKeyMissing(t *testing.T) {
	_, err := sign.LoadKey("/nonexistent/key.hex")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

//fusa:test REQ-FO-SIGN002
func TestLoadKeyBadHex(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "bad.hex")
	if err := os.WriteFile(keyPath, []byte("not-hex\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := sign.LoadKey(keyPath)
	if err == nil {
		t.Fatal("expected error for bad hex, got nil")
	}
}

//fusa:test REQ-FO-SIGN002
func TestLoadKeyTooShort(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "short.hex")
	// 15 bytes = 30 hex chars — below minimum of 16.
	if err := os.WriteFile(keyPath, []byte(strings.Repeat("ab", 15)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := sign.LoadKey(keyPath)
	if err == nil {
		t.Fatal("expected error for short key, got nil")
	}
}

//fusa:test REQ-FO-SIGN003
func TestSign(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	key, err := sign.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}

	artifact := filepath.Join(dir, "artifact.zip")
	if writeErr := os.WriteFile(artifact, []byte("fake zip content"), 0o640); writeErr != nil {
		t.Fatalf("WriteFile artifact: %v", writeErr)
	}

	sig, signErr := sign.Sign(artifact, key)
	if signErr != nil {
		t.Fatalf("Sign: %v", signErr)
	}
	if len(sig) != 64 {
		t.Errorf("signature length = %d hex chars, want 64", len(sig))
	}

	// Signature file should exist.
	sigPath := artifact + sign.SigExt
	if _, statErr := os.Stat(sigPath); statErr != nil {
		t.Errorf("signature file not created: %v", statErr)
	}
}

//fusa:test REQ-FO-SIGN003
func TestSignMissingFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	key, err := sign.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}
	_, err = sign.Sign(filepath.Join(dir, "missing.zip"), key)
	if err == nil {
		t.Fatal("expected error for missing artifact, got nil")
	}
}

//fusa:test REQ-FO-SIGN004
func TestVerify(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	key, err := sign.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}

	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("fake zip content"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, signErr := sign.Sign(artifact, key); signErr != nil {
		t.Fatalf("Sign: %v", signErr)
	}

	if err := sign.Verify(artifact, key); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

//fusa:test REQ-FO-SIGN004
func TestVerifyTamperedContent(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	if err := sign.Keygen(keyPath); err != nil {
		t.Fatalf("Keygen: %v", err)
	}
	key, err := sign.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}

	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("original content"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, signErr := sign.Sign(artifact, key); signErr != nil {
		t.Fatalf("Sign: %v", signErr)
	}

	// Tamper with the artifact.
	if err := os.WriteFile(artifact, []byte("tampered content"), 0o640); err != nil {
		t.Fatalf("WriteFile tampered: %v", err)
	}

	if err := sign.Verify(artifact, key); err == nil {
		t.Fatal("expected error for tampered artifact, got nil")
	}
}

//fusa:test REQ-FO-SIGN004
func TestVerifyMissingSig(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("content"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	key := make([]byte, 32)
	if err := sign.Verify(artifact, key); err == nil {
		t.Fatal("expected error for missing sig file, got nil")
	}
}

//fusa:test REQ-FO-SIGN004
func TestVerifyWrongKey(t *testing.T) {
	dir := t.TempDir()

	key1Path := filepath.Join(dir, "key1.hex")
	key2Path := filepath.Join(dir, "key2.hex")
	if err := sign.Keygen(key1Path); err != nil {
		t.Fatalf("Keygen key1: %v", err)
	}
	if err := sign.Keygen(key2Path); err != nil {
		t.Fatalf("Keygen key2: %v", err)
	}
	key1, _ := sign.LoadKey(key1Path)
	key2, _ := sign.LoadKey(key2Path)

	artifact := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(artifact, []byte("content"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := sign.Sign(artifact, key1); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if err := sign.Verify(artifact, key2); err == nil {
		t.Fatal("expected error for wrong key, got nil")
	}
}
