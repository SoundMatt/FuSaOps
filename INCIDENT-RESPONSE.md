# Incident Response Plan

This plan covers security and safety incidents affecting FuSaOps, in support of
IEC 62443-4-2 CR 6.2.1 and ISO 21434 incident-response expectations. Because
FuSaOps orchestrates other tools and can gate releases, an incident here may
mean unsafe code passes a gate undetected.

## Scope

An incident is any of:

- **Dropped or masked findings** — FuSaOps reports PASS while a bundled tool
  actually found (or should have found) an ERROR (hazards H-001, H-002).
- **Silent failure** — `fusaops check` exits 0 despite ERROR-severity findings
  (hazard H-003).
- **Supply-chain compromise** — a malicious or tampered x-FuSa tool image, or a
  compromised FuSaOps release/image.
- **Vulnerability** — a security defect in FuSaOps or its dependencies.

## Reporting

Report privately via GitHub Security Advisories on `SoundMatt/FuSaOps`, or email
the maintainers. Do not open public issues for security-sensitive reports. Target
acknowledgement: **5 working days**.

## Response process

1. **Triage** — confirm and classify the incident; assess whether any release
   gated by FuSaOps may have shipped unsafe code (check the affected version and
   the safety goals in `.fusa-hara.json`).
2. **Contain** — if a released image is affected, mark it and publish guidance;
   if a tool image is suspect, pin/withdraw it from the bundle.
3. **Fix** — land a regression test reproducing the incident first, then the fix.
   For dropped/masked findings, the test must assert the finding survives
   aggregation (SG-001/SG-002).
4. **Verify** — `go test -race ./...`, `gofusa check`, and `gofusa trace` clean;
   coverage at or above gate.
5. **Disclose** — publish a security advisory with affected versions, impact, and
   remediation. Bump the patch version and re-publish the image.
6. **Learn** — if a new failure mode was found, add it to `.fusa-hara.json` and a
   corresponding requirement + test.

## Severity and timelines

| Severity | Example | Target fix |
|---|---|---|
| Critical | Silent failure / dropped ERROR in a gating context | 72 hours |
| High | Masked coverage gap; supply-chain tampering | 7 days |
| Medium | Misattribution; dashboard data exposure | 30 days |
| Low | Cosmetic / non-gating | next release |

## Contacts

Maintainers: `SoundMatt` (GitHub). Security advisories:
<https://github.com/SoundMatt/FuSaOps/security/advisories>.
