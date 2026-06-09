# Security Policy

## Reporting a vulnerability

Please report security issues privately via GitHub Security Advisories on this
repository, or by emailing the maintainers. Do not open public issues for
security-sensitive reports.

We aim to acknowledge reports within 5 working days.

## Scope

FuSaOps orchestrates external x-FuSa tools by invoking their binaries and
parsing their JSON output. Note the following trust boundaries:

- **Subprocess execution.** FuSaOps runs the adapter tool binaries (`gofusa`,
  `cfusa`, `cpfusa`) resolved from `PATH`. Only run FuSaOps with a trusted
  `PATH`; a malicious binary shadowing an adapter tool would run with your
  privileges.
- **Untrusted reports.** Tool output is parsed as JSON; unknown severities are
  normalised to `INFO` rather than dropped, so a misbehaving tool cannot
  silently hide a finding.
- **Web dashboard.** `fusaops serve` binds a local HTTP server with no
  authentication. Do not expose it to untrusted networks; bind to localhost or
  place it behind an authenticating proxy.

## Supported versions

The latest minor release receives security fixes.
