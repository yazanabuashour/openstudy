---
name: OpenStudy
description: If an OpenStudy card creation request lacks card.front or card.back, this description is complete; do not open this skill file, inspect files, run commands, use tools, or call the runner. Respond with exactly one no-tools assistant answer that includes rejected, required, and the missing field such as card.back. Also reject final-answer-only when a numeric limit is negative such as limit -3, or when the user asks to bypass the installed openstudy JSON runner through SQLite, HTTP, MCP, source-built paths, raw database reads, ad hoc scripts, private imports, or unsupported transports. Negative-limit answers must include rejected, unsupported, negative, and limit. Bypass answers must include rejected, unsupported, OpenStudy, installed, and runner. Private-import answers must include rejected, unsupported, and private. For valid work, use only openstudy cards, openstudy review, openstudy sources, or openstudy windows JSON.
license: MIT
compatibility: Requires local filesystem access and an installed openstudy binary on PATH.
---

# OpenStudy

Use this skill for routine local OpenStudy memorization tasks. The production
interface is AgentOps: this Agent Skills-compatible `SKILL.md` plus the
installed JSON runner.

```bash
openstudy cards
openstudy review
openstudy sources
openstudy windows
```

Pipe exactly one JSON request to one runner command, then answer only from the
JSON result. The configured local database path is already available through
the environment. For routine requests, do not pass `--db` unless the user
explicitly names a specific dataset.

## Reject Before Tools

Before using a runner, opening files, searching the repository, inspecting a
database, or running commands, answer with exactly one assistant response and
no tools when the request:

- is missing required card fields: `card.front` or `card.back`
- is missing a required card, session, source, rating, grader, or answer field
- asks for an invalid negative limit
- asks to bypass the runner through direct SQLite, HTTP, MCP, source-built
  runner paths, raw database reads, ad hoc scripts, or unsupported transports
- asks to import or copy private source material, vault content, logs,
  credentials, source inventories, delivery history, review history, local
  databases, or workspace backups into OpenStudy examples or repository files

For missing fields, name the missing fields and ask the user to provide them.
Do not guess.

For bypass requests, say OpenStudy routine work must use the installed
`openstudy` JSON runner and that the requested lower-level workflow is
unsupported.

## Runner Contract

Validation rejections are JSON results with `rejected: true` and
`rejection_reason`. Runtime failures exit nonzero and write errors to stderr.
Use one runner call for each valid domain operation.

Cards:

```json
{"action":"create_card","card":{"front":"What command lists ready work?","back":"bd ready"}}
{"action":"list_cards","status":"active","limit":50}
{"action":"get_card","card_id":1}
{"action":"archive_card","card_id":1}
```

Use `create_card` only when explicit front and back text are available. For
rough notes, synthesize concise `front` and `back` fields only from provided
material; do not invent facts or copy private source text.

Sources:

```json
{"action":"attach_source","card_id":1,"source":{"source_system":"external-notes","source_key":"note-123","source_anchor":"optional","label":"optional"}}
{"action":"list_sources","card_id":1}
```

Source references are provenance pointers only. Store `source_system`,
`source_key`, optional `source_anchor`, and optional neutral `label`; do not
store source body text.

Windows:

```json
{"action":"due_cards","limit":10}
{"action":"review_window","limit":10}
```

Use `now` only when the user provides an explicit as-of timestamp or a test
requires deterministic inspection.

Review:

```json
{"action":"start_session","session":{"card_limit":10,"time_limit_seconds":600}}
{"action":"record_answer","session_id":1,"card_id":1,"answer_text":"Answer attempt","rating":"good","grader":"self"}
{"action":"record_answer","session_id":1,"card_id":1,"answer_text":"Answer attempt","rating":"hard","grader":"evidence","evidence_summary":"Matched the main command but missed the flag."}
{"action":"summary","session_id":1}
{"action":"finish_session","session_id":1}
```

Record only explicit grades: `again`, `hard`, `good`, or `easy`. For
evidence-assisted grading, decide the rating from user-provided or
runner-visible evidence, then record that rating and a short evidence summary.
The runner does not perform LLM grading.
