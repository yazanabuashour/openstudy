# OpenStudy

OpenStudy is being planned as a local-first AgentOps memorization system for
agents. The intended product owns memorization practice state: cards, review
scheduling, grading history, and automation state. Cards may later link back to
OpenClerk or vault source notes for provenance, but OpenStudy owns mutable
review practice data.

## Planning Status

This repository is in a planning-only stage. No product API, database schema,
runner contract, scheduler choice, skill contract, or implementation is
accepted until the Beads decision process explicitly promotes it.

OpenStudy uses two existing local projects as references:

- OpenHealth is the infrastructure reference: an installed JSON runner, a
  single-file skill, local SQLite storage, immutable release assets,
  repo-relative documentation, and production eval gates.
- OpenClerk is the decision-process reference: ADR, POC, eval, decision, then
  blocked implementation only after the decision promotes a surface.

## Privacy

Committed docs, reports, Beads notes, and artifact references must use
repo-relative paths or neutral placeholders. Do not commit machine-absolute
paths, credentials, private infrastructure details, personal data, raw private
logs, or sensitive sample content.

## Beads

Beads is initialized with the `os` prefix in embedded mode. In `bd 1.0.3`,
`bd doctor --json` reports that doctor is not yet supported for embedded mode;
use `bd status --json`, `bd list --json`, `bd ready --json`, `bd where`, and
`bd context --json` for routine verification.
