# Threat Analysis and Risk Assessment (TARA)

**Module:** github.com/SoundMatt/FuSaOps  
**Generated:** 2026-06-10T14:32:27Z  
**Standard:** ISO 21434 Chapter 9  

| ID | Asset | Threat | STRIDE | CWE | Vector | Likelihood | Impact | SL | Control | Residual Risk |
|---|---|---|---|---|---|---|---|---|---|---|
| TARA-001 | adapter.go | Command injection from variable input enables arbitrary command execution | E/R | CWE-78 | Network | Medium | High | 3 | Use exec.Command with fixed command and sanitised args | Low after remediation |
