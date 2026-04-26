# Contributing

Outside contributors do not need Beads to contribute to this repository.

## Project Shape

OpenStudy is in its initial promoted infrastructure stage. The repository does
not yet expose an `openstudy` runner, a skill, a database schema, a scheduler,
an install script, a release workflow, or an executable eval harness.

Future product surfaces must follow the accepted Beads decision chain before
they are added. In particular, storage and scheduling work belongs to the next
implementation bead, runner and skill behavior belongs to the runner/skill
bead, and eval or release gates belong to the eval/release bead.

## Local Setup

Maintainers prefer:

```bash
mise trust
mise install
```

Outside contributors may use their own tooling if they can satisfy the
repository checks. Beads and Dolt are maintainer-only tools and are not required
to open, review, or merge pull requests.

Current repository checks are:

```bash
git diff --check
mise exec -- go test ./...
```

If future code, runner, skill, eval, or release surfaces are promoted, update
this file with the new checks in the same change.

## Pull Request Expectations

- Keep changes reviewable without access to Beads state.
- Update repository docs when the public contract changes.
- Do not add private study material, vault content, source inventories, local
  databases, credentials, raw logs, workspace backups, delivery history, review
  history, or private infrastructure details.
- Route security issues through the private process in [SECURITY.md](SECURITY.md),
  not through public issues or pull requests.

## Support and Compatibility

Before `1.0`, compatibility is best effort and may change between planning and
implementation milestones. OpenStudy does not currently promise a shipped
runner, hosted service, remote HTTP API, MCP server, skill package, install
path, release artifact, or database schema.
