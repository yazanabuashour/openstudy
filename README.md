# OpenStudy

OpenStudy is a local-first AgentOps memorization runtime for agents. The
intended product helps agents practice and retain operational
knowledge through memorization workflows, and it owns memorization practice
state: cards, review scheduling, grading history, and automation state. Cards
may later link back to OpenClerk or vault source notes for provenance, but
OpenStudy owns mutable review practice data.

OpenStudy is designed for open-source distribution. This repository must not
contain personal source inventories, private study material, private vault
content, delivery or review logs, workspace backups, run history, local SQLite
databases, credentials, private infrastructure details, raw private logs, or
sensitive sample content.

## Planning Status

This repository has completed the first planning decision chain. Decision
[`docs/decision/0001-openstudy-memorization-promotion.md`](docs/decision/0001-openstudy-memorization-promotion.md)
accepted a narrow implementation path. The `os-ful` bead adds the first
internal storage and scheduler layer, and `os-5v4` adds the first JSON runner
and single-file skill. Install scripts, release workflow, automation runtime,
and executable eval harnesses remain gated behind ordered Beads implementation
work.

The current planning ADR is
[`docs/adr/0001-agentops-memorization-direction.md`](docs/adr/0001-agentops-memorization-direction.md).
It frames the AgentOps memorization direction without promoting implementation.
The current planning POC is
[`docs/poc/0001-memorization-architecture-options.md`](docs/poc/0001-memorization-architecture-options.md).
It compares architecture options without shipping product code.
The current eval plan is
[`docs/eval/0001-agentops-memorization-pressure.md`](docs/eval/0001-agentops-memorization-pressure.md).
It defines pressure scenarios without adding an eval harness.
The accepted promotion decision is
[`docs/decision/0001-openstudy-memorization-promotion.md`](docs/decision/0001-openstudy-memorization-promotion.md).
It promotes an OpenHealth-style path while keeping implementation ordered
through the Beads placeholder chain.

OpenStudy uses two existing local projects as references:

- OpenHealth is the infrastructure reference: an installed JSON runner, a
  single-file skill, local SQLite storage, immutable release assets,
  repo-relative documentation, and production eval gates.
- OpenClerk is the decision-process reference: ADR, POC, eval, decision, then
  blocked implementation placeholders until the accepted decision promotes a
  surface.
- OpenBrief is the documentation and repository-hygiene reference for
  open-source distribution, local runtime state boundaries, and keeping private
  user configuration out of the repository.

## AgentOps Direction

The agent-facing path is the AgentOps pattern: a single-file skill gives the
agent task policy, and a local JSON runner performs stateful memorization
operations through structured JSON.

## Runner Interface

OpenStudy exposes an `openstudy` JSON runner for local use:

```bash
openstudy cards
openstudy review
openstudy sources
openstudy windows
```

Each domain accepts one JSON request on stdin and returns one JSON response on
stdout. Validation failures return JSON with `rejected: true`; runtime failures
exit nonzero and write to stderr. The single-file agent skill is
[`skills/openstudy/SKILL.md`](skills/openstudy/SKILL.md).

## Internal Local Storage

OpenStudy is local-first. Mutable memorization state lives in a host-local
SQLite database outside the repository, following the OpenHealth and OpenBrief
pattern. Internal runtime path resolution uses
`${XDG_DATA_HOME:-~/.local/share}/openstudy/openstudy.sqlite`, with
`OPENSTUDY_DATABASE_PATH` and explicit config overrides for tests and future
runner wiring. The database remains an implementation detail, not a routine
agent control plane.

## Development

Current implementation work is limited to internal storage, scheduling, runner,
skill, docs, Beads state, and repository infrastructure until later beads
promote eval, release, installation, and automation surfaces.

Use the pinned local toolchain for repository development:

```bash
mise trust
mise install
mise exec -- go test ./...
```

Useful verification commands for this stage:

```bash
git diff --check
mise exec -- bd status --json
mise exec -- bd list --json
mise exec -- bd ready --json
mise exec -- bd dep cycles --json
mise exec -- bd where
mise exec -- bd context --json
```

Beads is initialized with the `os` prefix in embedded mode. In `bd 1.0.3`,
`bd doctor --json` reports that doctor is not yet supported for embedded mode;
use the commands above for routine verification.

## Future Releases

No OpenStudy release assets exist yet, and runner/skill work must not add any.
If a release process is promoted later, tagged `v0.y.z` releases should follow
the OpenHealth and OpenBrief posture: platform binary archives, skill archives,
installer assets, source archives, checksums, SBOMs, attestations, release
verification docs, and immutable published assets.

## Contributing

Outside contributors should be able to work through GitHub issues and pull
requests once contribution docs exist. Beads is maintainer-only workflow tooling
and is not required for community contributions.
