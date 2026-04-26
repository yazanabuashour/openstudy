# OpenStudy

OpenStudy is being planned as a local-first AgentOps memorization runtime for
agents. The intended product helps agents practice and retain operational
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
accepts a narrow future implementation path, but product behavior is still
gated behind ordered Beads implementation work. No product API, database schema,
runner behavior, scheduler implementation, skill contract, install script,
release workflow, or executable eval harness is present in this repository yet.

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

The expected agent-facing path is the AgentOps pattern: a single-file skill
gives the agent task policy, and a local JSON runner performs stateful
memorization operations through structured JSON. That shape is not implemented
yet. The exact runner domains, request and response schema, validation model,
storage behavior, scheduler, and automation surface must be decided through the
ADR, POC, eval, and decision beads before any code is added.

## Deferred Runner Interface

OpenStudy does not currently ship an `openstudy` runner, skill, install path, or
public command surface. Candidate domains from the planning work may include
deck/card management, review sessions, answer recording, grading evidence, and
review-window automation, but those names are placeholders until promoted by a
decision bead and implemented by the relevant follow-up issue.

## Deferred Local Storage

OpenStudy is expected to be local-first. If promoted, mutable memorization state
should live in a host-local database outside the repository, following the
OpenHealth and OpenBrief pattern. The database path, environment variables,
schema, migrations, backup expectations, and direct-access restrictions are
deferred until the storage and runner decisions are accepted.

## Development

There is no product implementation yet. Current development work is limited to
docs, Beads planning state, and the initial repository infrastructure scaffold.

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

No OpenStudy release assets exist yet, and this planning task must not add any.
If a release process is promoted later, tagged `v0.y.z` releases should follow
the OpenHealth and OpenBrief posture: platform binary archives, skill archives,
installer assets, source archives, checksums, SBOMs, attestations, release
verification docs, and immutable published assets.

## Contributing

Outside contributors should be able to work through GitHub issues and pull
requests once contribution docs exist. Beads is maintainer-only workflow tooling
and is not required for community contributions.
