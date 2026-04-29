# OpenStudy Taste Review Backlog

## Status

Planning backlog created after `os7nh`.

This note records a process correction, not a new public API. It keeps the
successful ADR, POC, eval, decision, implementation, and release-gate workflow,
while adding a clearer taste review for cases where OpenStudy is technically
safe but unnecessarily awkward.

## Baseline Lesson

`os7nh` is the baseline release-blocking eval for the installed runner and
skill. The reduced report shows that the production surface passed all eight
scenarios while also exposing where routine workflows may be expensive or
ceremonial:

- `rough-card-create` completed through runner-mediated card creation and
  provenance attachment.
- `source-provenance` completed with neutral source pointer fields only.
- `due-window-review` and `scheduler-transition` completed with higher command
  and assistant-turn counts than the rejection scenarios.
- missing-field, negative-limit, bypass, and private-data cases correctly
  rejected before tools.

Evidence:

- `docs/evals/agent-production.md`
- `docs/evals/results/os7nh-v0.1.0.md`
- `docs/eval/0001-agentops-memorization-pressure.md`
- `skills/openstudy/SKILL.md`

## Taste Review Lens

Future deferral, reference, or pass decisions should ask one more question
after the safety and capability checks: would a routine agent or maintainer
reasonably expect a simpler OpenStudy surface here?

Useful signals include:

- the workflow passes but needs many runner calls, assistant turns, or exact
  prompt choreography.
- the user intent fits the natural scope of an existing runner domain, but the
  current workflow requires surprising manual decomposition.
- the agent asks for approval before a read, fetch, or inspect step when the
  real approval boundary is a durable write, credentialed access, external
  mutation, or irreversible action.
- the result is safe but ceremonial, high latency, or hard to explain to a
  routine user.

This lens does not weaken OpenStudy invariants. Provenance, authority,
auditability, local-first runner-only access, privacy, explicit approval
boundaries, repository hygiene, bypass rejection, and immutable release gates
still decide whether a smoother surface is acceptable.

## Tracker Backlog

The Beads backlog should track four audit/design/eval epics:

- Re-audit card intake and source acquisition UX.
- Re-audit naming, placement, and card organization UX.
- Re-audit high-touch successful review workflows.
- Update OpenStudy decision process for taste.

These epics do not authorize runner actions, schema changes, storage
migrations, skill behavior changes, public APIs, product behavior changes, or
implementation follow-up. Any future implementation still needs targeted
evidence and an explicit promotion decision naming the exact surface and gates.

## Initial Audit Targets

Re-audit rough-card intake and source pointer capture after `os7nh`. Treat
runner-mediated creation with provenance pointers as the safe baseline, then
compare natural rough-note capture, source hints, and approval boundaries for
remaining places where agents may be forced into ceremony.

Re-audit naming, placement, and card organization around exact `front`, `back`,
source label, card identifier, and session identifier requirements. The
question is when OpenStudy should infer, propose, or ask for card wording,
source labels, future deck or group placement, and source hints.

Re-audit high-touch successful workflows where natural tasks passed but stayed
expensive or brittle. Initial candidates include `due-window-review`,
`scheduler-transition`, evidence-assisted grading, review summaries, and
strict missing-field rejection paths that may require surprising clarification
turns.

Update process docs so future eval reports can separately record:

- safety pass: the workflow preserved invariants and rejected bypasses.
- capability pass: current primitives can technically express the workflow.
- UX quality: the workflow is or is not acceptable for routine use.
