# Project Instructions for AI Agents

This file provides instructions and context for AI coding agents working on this project.

## OpenStudy Planning Boundary

OpenStudy is currently planning-only. Do not add a runner, skill, database
schema, scheduler implementation, automation runtime, or product API until the
Beads ADR, POC, eval, and decision chain explicitly promotes that work.

Use OpenHealth as the infrastructure reference: installed JSON runner,
single-file skill, local SQLite storage, immutable release posture,
repo-relative docs, and eval gates. Use OpenClerk as the decision-process
reference: ADR, POC, eval, decision, then blocked implementation placeholders.

## Privacy And Artifacts

For committed docs, reports, Beads notes, and artifact references, use
repo-relative paths or neutral placeholders. Never commit machine-absolute
filesystem paths, credentials, private infrastructure details, personal data,
raw private logs, or sensitive sample content.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->


## Build & Test

There is no product implementation yet. For the current planning-only stage,
verify docs and Beads state with `git diff --check`, `bd status --json`, and
the Beads graph checks described in active issues.

## Architecture Overview

Architecture is intentionally deferred until the ADR, POC, eval, and decision
chain promotes an implementation surface.

## Conventions & Patterns

Keep project work local-first, AgentOps-oriented, repo-relative, and
privacy-preserving.
