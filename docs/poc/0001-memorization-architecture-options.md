# POC 0001: Memorization Architecture Options

Status: Planning evidence

## Context

This POC compares candidate OpenStudy memorization architecture options without
shipping product code. It builds on
[`docs/adr/0001-agentops-memorization-direction.md`](../adr/0001-agentops-memorization-direction.md),
which frames OpenStudy as the future owner of mutable memorization practice
state while treating external source references as provenance only.

This document is not an implementation spec. It does not define a public API,
runner command, skill contract, database schema, scheduler implementation,
automation runtime, install path, release workflow, or eval harness. Candidate
names below are planning labels only until the eval and decision beads promote
a surface.

## Scheduler Candidates

FSRS-style scheduling is the strongest primary candidate for later eval because
it is designed for adaptive review intervals and can model memory state as
practice evidence accumulates. It likely fits OpenStudy's goal better than a
fixed queue, but it also carries higher complexity, more state to validate, and
more pressure on eval coverage.

SM-2 is the fallback candidate. It is easier to explain and evaluate, and it
may be sufficient if the first promoted surface needs a smaller scheduling
model. Its weakness is lower adaptability and less room to represent nuanced
free-text grading evidence.

Manual queueing is the baseline candidate. It avoids scheduler complexity and
is useful for deterministic operator-controlled review windows, but it does not
meet the full memorization-runtime goal by itself.

POC recommendation: carry FSRS-style scheduling forward as the primary eval
candidate, keep SM-2 as fallback, and use manual queueing as a baseline for
comparison.

## Runner Domains

If later promoted, the runner should remain local-first: an installed local JSON
runner invoked by a single-file skill. Candidate runner domains are:

- decks/cards: create and maintain practice items as OpenStudy-owned state.
- review sessions: select due or manually queued cards for a bounded session.
- answer recording: capture answer attempts without importing private source
  content.
- grading evidence: record self-grade or evidence-assisted results.
- source references: store provenance pointers to external sources without
  copying private material.
- review windows: expose due-review windows for automation planning without
  adding a scheduler runtime in this POC.

These are candidate domains only. Exact request shapes, response shapes,
validation rules, command names, and persistence behavior remain deferred to
the decision bead.

POC recommendation: prefer a runner-mediated interface over direct library,
SQLite, HTTP, MCP, source-built, or ad hoc script access so future skills can
enforce policy and reject bypasses.

## Grading Workflow

Self-grade is the simplest candidate: an agent or operator records a rating
after seeing the answer. It is easy to validate and explain, but quality depends
on consistent human or agent judgment.

Evidence-assisted free-text grading is the higher-value candidate. It would let
OpenStudy compare a free-text answer with expected evidence or source
references while still recording structured practice results. This needs eval
pressure because it can fail through overgenerous grading, hallucinated
evidence, or accidental private-content exposure.

Explicit missing-field rejection is required for any promoted workflow. Future
runner calls should reject incomplete card, review, answer, grading, or source
reference inputs rather than guessing.

POC recommendation: evaluate self-grade as the minimum viable workflow,
pressure-test evidence-assisted free-text grading, and require missing-field
rejection in the eval plan.

## Source References

OpenStudy should store provenance references to external sources, not source
content. Candidate references may identify the source system, stable source key,
optional section or anchor, and a neutral label. They must not copy private
study material, vault text, delivery logs, review logs, source inventories,
credentials, or local filesystem details into this repository.

POC recommendation: treat source references as lightweight provenance pointers
owned by OpenStudy records. Private content remains outside OpenStudy docs and
outside the open-source repository.

## Automation Windows

Automation windows are planning candidates for when reviews should be offered,
not a scheduler implementation. Candidate models include:

- due-now windows for cards whose scheduling evidence says they are ready.
- bounded review sessions capped by count or time budget.
- manual focus windows where an operator asks for review material now.
- quiet windows where automation should avoid prompting.

POC recommendation: carry review-window modeling into the eval as observable
planning behavior while deferring automation runtime, reminders, monitors, and
scheduling implementation.

## Local-First Fit

The established local-first release posture remains the infrastructure
reference if implementation is later promoted: installed JSON runner,
single-file skill, host-local storage outside the repository, repo-relative
documentation, immutable release assets, and eval gates before release.

This shape fits OpenStudy because memorization state is mutable and local, the
skill can provide task policy to agents, and the runner can centralize
validation, persistence, privacy checks, and bypass rejection. It also keeps
future releases verifiable instead of relying on source-built local behavior.

POC recommendation: keep local-first runner infrastructure as the preferred
path for any promoted implementation.

## Risks and Eval Carry-Forward

The next eval should pressure-test:

- rough-term card creation without requiring polished source notes.
- missing-field rejection across candidate runner domains.
- free-text grading quality and evidence handling.
- due review sessions and scheduler transitions.
- source-reference provenance without private-content import.
- rejection of direct SQLite, source-built, HTTP, MCP, or unsupported bypasses.
- private-data redaction in docs, outputs, and fixture-like examples.

## Recommendation

Carry forward a local JSON runner plus single-file skill as the preferred
future runtime shape, with OpenStudy-owned host-local mutable state and
provenance references only. Evaluate FSRS-style scheduling first, keep SM-2 as
fallback, and retain manual queueing as a baseline. Require bypass rejection and
private-data redaction as acceptance criteria for the eval and decision chain.

This POC does not authorize implementation. Final promotion remains deferred to
`os-24u` and `os-pke`.
