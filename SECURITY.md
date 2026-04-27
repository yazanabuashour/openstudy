# Security Policy

## Supported Versions

This project is pre-`1.0` and currently contains planning documents, internal
storage/scheduler code, and the first OpenStudy JSON runner plus single-file
skill. It also contains local production eval and release verification tooling.
The first planned public release is `v0.1.0`. There is no hosted service.

The supported code line is the current default branch until release artifacts
exist.

## Reporting a Vulnerability

Do not report vulnerabilities in public issues, pull requests, discussions, or
Beads notes.

Use GitHub private vulnerability reporting from the repository Security tab.
Include:

- a clear description of the issue
- affected files or workflow surfaces
- reproduction steps or proof-of-concept details
- expected impact and known mitigations

If GitHub private reporting is unavailable, contact the repository owner through
an existing private channel and share only enough detail to arrange private
handoff. Do not disclose the vulnerability publicly while that handoff is being
arranged.

## Response Expectations

These are targets, not contractual guarantees:

| Severity | Initial acknowledgment | Status update target | Patch or mitigation target |
| --- | --- | --- | --- |
| Critical | within 2 business days | within 5 calendar days | within 14 calendar days |
| High | within 3 business days | within 7 calendar days | within 30 calendar days |
| Medium | within 5 business days | within 14 calendar days | next planned release or documented mitigation |
| Low | within 5 business days | as needed | next routine release if accepted |

## Severity Handling

Maintainers triage reports using practical impact on repository users and
maintainers:

- Critical: repository compromise, credential exposure, arbitrary code
  execution in trusted automation, or release-integrity failure.
- High: meaningful integrity or privilege risk without a full repository
  compromise.
- Medium: exploitable weakness with limited blast radius or clear
  prerequisites.
- Low: hard-to-exploit issue, defense-in-depth gap, or low-impact
  misconfiguration.

## Patch and Advisory Process

- Fixes land privately first when needed to avoid widening exposure.
- Public release notes should avoid exploit-enabling detail until a fix or
  mitigation is available.
- If the repository later adopts GitHub Security Advisories, maintainers should
  publish advisories for material fixes.

## Emergency Releases and Hotfixes

If a vulnerability affects a future supported release, maintainers may cut an
out-of-band patch tag and hosted release outside the normal release cadence.

Any future emergency release must follow the immutable release posture: pass
production eval gates, verify checksums and attestations, and ship a new patch
release instead of replacing published assets.
