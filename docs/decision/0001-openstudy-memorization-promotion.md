# Decision 0001: OpenStudy Memorization Promotion

Status: Accepted / narrow promotion

Bead: `os-pke`

## Context

OpenStudy has completed the current planning chain for AgentOps memorization:

- [`docs/adr/0001-agentops-memorization-direction.md`](../adr/0001-agentops-memorization-direction.md)
- [`docs/poc/0001-memorization-architecture-options.md`](../poc/0001-memorization-architecture-options.md)
- [`docs/eval/0001-agentops-memorization-pressure.md`](../eval/0001-agentops-memorization-pressure.md)

The ADR frames OpenStudy as the future owner of mutable memorization practice
state. The POC recommends an OpenHealth-style local JSON runner plus
single-file skill. The eval plan defines pressure gates for card creation,
missing-field rejection, grading quality, due reviews, scheduler transitions,
provenance boundaries, bypass rejection, and private-data redaction.

This decision promotes a minimal implementation path for future beads only. It
does not add a runner, skill, database schema, scheduler implementation,
automation runtime, product API, install script, release workflow, or executable
eval harness.

## Decision

Promote an OpenHealth-style OpenStudy runtime path behind the existing ordered
implementation placeholders. OpenStudy should proceed through the blocked Beads
chain rather than implementing product behavior directly from this decision:

1. `os-560`: scaffold promoted OpenStudy infrastructure.
2. `os-ful`: implement promoted storage and scheduler.
3. `os-5v4`: implement promoted runner and skill.
4. `os-7nh`: add promoted eval and release gates.

The future public agent-facing surface is:

- installed runner binary: `openstudy`
- single-file skill: `skills/openstudy/SKILL.md`
- transport: one structured JSON request on stdin and one structured JSON
  response on stdout
- validation rejection shape: structured JSON with `rejected: true`
- runner domains: `openstudy cards`, `openstudy review`, `openstudy sources`,
  and `openstudy windows`

The initial action families are:

- card create, list, get, and archive
- due-card selection
- review session start, record, and summary
- self-grade and evidence-assisted grade record
- provenance pointer attach and list
- review-window inspection

Exact request fields, response fields, schema migrations, command flags, and
error codes remain implementation-bead responsibilities, but they must preserve
this promoted surface and the gates below.

## Storage and Scheduler

OpenStudy owns mutable memorization practice state. The promoted storage path is
host-local SQLite outside the repository. The default database location is
`${XDG_DATA_HOME:-~/.local/share}/openstudy/openstudy.sqlite`, overridable by
`OPENSTUDY_DATABASE_PATH` or `--db`.

The SQLite database is an implementation detail of the runner. Routine agents
must not query it directly, and direct SQLite workflows must be rejected rather
than documented as an alternate control plane.

FSRS-style scheduling is the promoted primary scheduler. SM-2 remains a fallback
only if later eval evidence shows the FSRS-style path is too complex or brittle
for the first production surface. Manual queueing may be added as an
operator-controlled overlay, but it is not the memorization scheduler.

## Gates

Future implementation must reject incomplete inputs instead of guessing across
card, review, grading, source-reference, and review-window operations.

Future implementation and docs must reject or avoid:

- direct SQLite access for routine agents
- HTTP, MCP, source-built runner paths, and unsupported transports as hidden
  alternate control planes
- copied private study material, vault content, delivery or review logs,
  workspace backups, raw private logs, credentials, local databases, private
  infrastructure details, and sensitive sample content
- machine-absolute filesystem paths in committed docs, reports, Beads notes, or
  artifact references

Every future OpenStudy eval scenario must run with `gpt-5.4-mini`. No eval
harness, promoted eval workflow, or release gate is acceptable unless that model
pin is explicit.

Any release path must follow the OpenHealth immutable-release posture: shipped
runner and skill artifacts, release verification, checksums or attestations as
appropriate for the artifact set, and passing production eval gates before
publication.

## Consequences

`os-pke` authorizes the future implementation chain, but only within the
promoted boundaries above. The current change remains documentation and tracker
state only.

Implementation beads may now refine concrete schemas, validation rules,
storage migrations, scheduler state transitions, runner commands, skill policy,
eval harness mechanics, install assets, and release workflows. Those later
changes must remain local-first, runner-mediated, privacy-preserving, and
eval-gated.

No product behavior is accepted until the relevant implementation bead lands
with its own tests and gates.
