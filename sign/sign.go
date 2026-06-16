// Package sign provides HMAC-SHA256 file signing for FuSaOps artifacts.
//
// Use Keygen to create a key, Sign to sign an artifact, and Verify to check
// it. All operations are stdlib-only and produce no external dependencies.
package sign

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SigExt is the extension appended to a file path to get its signature path.
//
//fusa:req REQ-FO-SIGN001
const SigExt = ".sig"

// Keygen generates a random 32-byte HMAC key, hex-encodes it, and writes it
// to path with mode 0o600. The path must not already exist.
//
//fusa:req REQ-FO-SIGN001
func Keygen(path string) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("sign: generate key: %w", err)
	}
	encoded := hex.EncodeToString(key) + "\n"
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		return fmt.Errorf("sign: write key %s: %w", path, err)
	}
	return nil
}

// LoadKey reads and hex-decodes a key file written by Keygen.
// Returns an error if the file is missing, not valid hex, or shorter than 16 bytes.
//
//fusa:req REQ-FO-SIGN002
func LoadKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sign: read key %s: %w", path, err)
	}
	// Strip trailing whitespace.
	for len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r' || data[len(data)-1] == ' ') {
		data = data[:len(data)-1]
	}
	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("sign: decode key hex: %w", err)
	}
	if len(key) < 16 {
		return nil, fmt.Errorf("sign: key too short: %d bytes (minimum 16)", len(key))
	}
	return key, nil
}

// Sign computes the HMAC-SHA256 of the file at path using key, writes it as
// hex to path+".sig" with mode 0o640, and returns the hex-encoded signature.
//
//fusa:req REQ-FO-SIGN003
func Sign(path string, key []byte) (string, error) {
	sig, err := hmacFile(path, key)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	encoded := hex.EncodeToString(sig) + "\n"
	sigPath := path + SigExt
	if err := os.WriteFile(sigPath, []byte(encoded), 0o640); err != nil {
		return "", fmt.Errorf("sign: write %s: %w", sigPath, err)
	}
	return hex.EncodeToString(sig), nil
}

// Verify checks that path+".sig" contains the correct HMAC-SHA256 for path.
// Returns nil on match, an error on mismatch or if the signature file is missing.
//
//fusa:req REQ-FO-SIGN004
func Verify(path string, key []byte) error {
	sigPath := path + SigExt
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("sign: read signature %s: %w", sigPath, err)
	}
	// Strip trailing newline.
	for len(sigData) > 0 && (sigData[len(sigData)-1] == '\n' || sigData[len(sigData)-1] == '\r') {
		sigData = sigData[:len(sigData)-1]
	}
	want, err := hex.DecodeString(string(sigData))
	if err != nil {
		return fmt.Errorf("sign: decode signature: %w", err)
	}
	got, err := hmacFile(path, key)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	if !hmac.Equal(got, want) {
		return fmt.Errorf("sign: signature INVALID for %s", path)
	}
	return nil
}

func hmacFile(path string, key []byte) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	h := hmac.New(sha256.New, key)
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("hash %s: %w", path, err)
	}
	return h.Sum(nil), nil
}
