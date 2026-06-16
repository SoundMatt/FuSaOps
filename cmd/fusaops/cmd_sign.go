package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/sign"
)

// runSign signs or verifies a file using HMAC-SHA256.
//
//fusa:req REQ-FO-CLI063
func runSign(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops sign", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops sign [flags] <file>\n\n")
		fmt.Fprintf(stderr, "Sign or verify a file using HMAC-SHA256.\n\n")
		fmt.Fprintf(stderr, "  fusaops sign --key keyfile artifact.zip           # creates artifact.zip.sig\n")
		fmt.Fprintf(stderr, "  fusaops sign --verify --key keyfile artifact.zip  # verifies artifact.zip.sig\n")
		fmt.Fprintf(stderr, "  fusaops sign --keygen keyfile                     # generate a new key\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		keyFile  = fs.String("key", "", "path to HMAC key file (32-byte hex)")
		doVerify = fs.Bool("verify", false, "verify an existing signature instead of creating one")
		keygen   = fs.String("keygen", "", "generate a new random key and write to this path")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *keygen != "" {
		if err := sign.Keygen(*keygen); err != nil {
			fmt.Fprintf(stderr, "fusaops sign: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Key written to %s (keep this secret)\n", *keygen)
		return 0
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	target := fs.Arg(0)

	if *keyFile == "" {
		fmt.Fprintf(stderr, "fusaops sign: --key is required\n")
		return 2
	}
	key, err := sign.LoadKey(*keyFile)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops sign: %v\n", err)
		return 1
	}

	if *doVerify {
		verifyErr := sign.Verify(target, key)
		if verifyErr != nil {
			fmt.Fprintf(stderr, "fusaops sign: %v\n", verifyErr)
			return 1
		}
		fmt.Fprintf(stdout, "Signature OK for %s\n", target)
		return 0
	}

	sig, signErr := sign.Sign(target, key)
	if signErr != nil {
		fmt.Fprintf(stderr, "fusaops sign: %v\n", signErr)
		return 1
	}
	fmt.Fprintf(stdout, "Signature written to %s%s\n", target, sign.SigExt)
	_ = sig
	return 0
}
