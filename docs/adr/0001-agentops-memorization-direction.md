# ADR 0001: AgentOps Memorization Direction

Status: Proposed / planning

## Context

OpenStudy is being planned as a local-first AgentOps memorization runtime for
agents. It is not an implementation repository yet: no runner, skill, database
schema, scheduler implementation, automation runtime, product API, install
script, release workflow, or eval harness is promoted by this ADR.

The planning chain needs a direction before the POC can compare architecture
options. This ADR frames that direction while keeping exact schemas, command
names, runner domains, grading behavior, scheduler selection, and release
surfaces deferred to the POC, eval, and decision beads.

## Decision

OpenStudy should be treated as the future owner of mutable memorization
practice state. External source references should be provenance inputs, not the
place where review state is stored or mutated.

If implementation is later promoted, the runtime should follow a local-first
release posture: an installed local JSON runner, a single-file agent skill,
host-local storage outside the repository, repo-relative documentation,
immutable release assets, and eval gates before release. This ADR does not
choose the final runner contract, skill contract, storage schema, scheduler, or
public command surface.

## Comparison

### Practice State

OpenStudy-owned state keeps memorization behavior independent from source-note
systems. Cards, due dates, grading history, and automation state can evolve as
practice data without rewriting external source notes.

Using external source content as the mutable review store would blur provenance
with practice state. It also risks leaking private study material, delivery
history, or source inventories into an open-source repository.

Direction: OpenStudy owns mutable practice state; external source pointers are
candidate provenance references only.

### Runner and Skill Shape

A local JSON runner plus single-file skill gives agents a narrow policy surface
and keeps stateful operations behind structured local commands. It also
supports eval gates and release verification before any behavior is published.

Direct library access, product APIs, or ad hoc scripts would make bypasses
easier, spread policy across implementation details, and promote surfaces before
the decision chain has accepted them.

Direction: prefer a future installed JSON runner and single-file skill, pending
POC evidence and explicit decision approval.

### Local Storage

Host-local SQLite-style storage fits the local-first goal and keeps mutable
practice data outside the repository. Database paths, environment variables,
backup expectations, migrations, and direct-access restrictions remain
undecided.

Repo-stored databases, private study exports, or fixture-like private samples
would conflict with the repository-hygiene boundary for open-source
distribution.

Direction: prefer host-local storage outside the repo if implementation is
promoted; do not add a schema in this ADR.

### Scheduler Options

FSRS-style scheduling is a strong candidate for adaptive memorization practice
and should be compared in the POC. SM-2 is a simpler fallback candidate that may
be useful if the POC favors a smaller dependency or easier explainability.
Manual queueing is a baseline candidate for workflows that need deterministic
operator control or minimal state transitions.

Direction: defer scheduler selection to the POC and eval. FSRS-style, SM-2, and
manual queueing remain planning candidates only.

### Privacy and Bypass Prevention

Future operations should be runner-mediated so the skill can reject unsupported
transports, direct database access, raw private logs, and source-built bypasses.
Private source inventories, study material, vault content, review logs, local
databases, credentials, and sensitive samples must stay out of the repository.

Direction: bypass prevention and private-data redaction are required design
constraints for the POC and eval plan, not optional polish.

## Non-Goals

- Define a public API, runner command, skill contract, database schema, install
  path, release workflow, or eval harness.
- Choose the final scheduler or grading algorithm.
- Import or model private study material, vault content, review history, logs,
  or local databases.
- Implement automation, review sessions, card management, grading, or source
  synchronization.
- Promote any placeholder implementation bead before the decision bead accepts
  the surface.

## Follow-Up

The next POC should compare the candidate runner domains, grading workflow,
source-reference model, scheduler options, automation windows, and local-first
infrastructure fit without shipping product code. The later eval should
pressure test missing-field rejection, free-text grading, due review sessions,
scheduler transitions, provenance, direct-bypass rejection, and private-data
redaction.
