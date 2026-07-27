# `fusaops sign`

Sign or verify a file using HMAC-SHA256 — for attesting the integrity of
release artefacts (SBOMs, audit packs, reports).

```bash
fusaops sign --key <keyfile> <file>            # creates <file>.sig
fusaops sign --verify --key <keyfile> <file>   # verifies <file>.sig
fusaops sign --keygen <keyfile>                # generate a new key
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--key` | — | Path to HMAC key file (32-byte hex) |
| `--verify` | off | Verify an existing signature instead of creating one |
| `--keygen` | — | Generate a new random key and write it to this path |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Signature created/verified successfully, or key generated |
| 1 | Missing/invalid key, or signature verification failed |
| 2 | Usage error (no file argument, missing `--key`) |

## Behaviour

- `--keygen` writes a fresh random key and exits; keep the key file secret —
  anyone with it can forge signatures.
- Signing writes `<file>.sig` alongside the target file.
- Verification recomputes the HMAC and compares against the `.sig` file.

## Example

```bash
fusaops sign --keygen release.key
fusaops sign --key release.key audit-pack.zip
fusaops sign --verify --key release.key audit-pack.zip
```
