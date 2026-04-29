# OpenStudy Domain Context

OpenStudy is a local-first AgentOps memorization runtime for agents. It owns
mutable memorization practice state while source systems remain provenance
inputs only.

## Domain Terms

- **Card**: an OpenStudy-owned practice item with explicit front and back text.
  Cards are mutable review practice state, not copied source notes.
- **Source reference**: a lightweight provenance pointer attached to a card.
  It stores a source system, stable source key, optional anchor, and optional
  neutral label. It does not store source body text.
- **Review window**: an inspection of cards due at a point in time, bounded by
  a caller-provided limit. A window is planning output, not an automation
  runtime.
- **Review session**: a bounded practice session created before answers are
  recorded. A session may have a card limit or time limit.
- **Review attempt**: a recorded answer for one card within a review session,
  with explicit answer text, rating, grader, and optional evidence summary.
- **Rating**: the explicit review outcome recorded as `again`, `hard`, `good`,
  or `easy`.
- **Grader**: the explicit grading mode recorded as `self` or `evidence`. The
  runner records grades; it does not perform LLM grading.
- **Card schedule**: FSRS-style scheduling state owned by OpenStudy for a card.
  It is persisted locally and updated by review attempts.
- **Scheduler transition**: the before-and-after card schedule produced by a
  review attempt. It is returned as behavior at the study module interface.
- **Runner-mediated access**: routine stateful work goes through the installed
  `openstudy` JSON runner and single-file skill. Direct SQLite, HTTP, MCP,
  source-built runner paths, ad hoc scripts, and unsupported transports are not
  alternate control planes.
- **Taste review**: a decision-process check that records safety pass,
  capability pass, and UX quality separately. Taste review can identify
  ceremonial or high-touch workflows, but it does not bypass provenance,
  authority, auditability, privacy, runner-only access, or release gates.

## Product Constraints

- OpenStudy remains local-first: installed JSON runner, single-file skill,
  host-local SQLite storage outside the repository, and release/eval gates.
- The SQLite database is an implementation detail of the runner.
- Public runner domains are `openstudy cards`, `openstudy review`,
  `openstudy sources`, and `openstudy windows`.
- Product surface changes require the established ADR, POC, eval, decision,
  and Beads promotion chain before implementation.
