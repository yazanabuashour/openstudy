# Contributing

Outside contributors do not need Beads to contribute to this repository.

## Project Shape

OpenStudy has internal storage and scheduler code from `os-ful`, plus the
promoted `openstudy` JSON runner and single-file skill from `os-5v4`. The
repository now includes production eval and local release verification tooling
from `os-7nh`, but it does not publish releases or expose automation runtime
surfaces.

Future product surfaces must follow the accepted Beads decision chain before
they are added.

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
mise exec -- ./scripts/validate-agent-skill.sh
mise exec -- ./scripts/validate-committed-artifacts.sh
mise exec -- ./scripts/validate-release-docs.sh
```

Production eval runs use:

```bash
mise exec -- go run ./scripts/agent-eval/os7nh run
```

Live eval execution requires local Codex CLI access and writes raw run
artifacts outside committed docs.

## Pull Request Expectations

- Keep changes reviewable without access to Beads state.
- Update repository docs when the public contract changes.
- Do not add private study material, vault content, source inventories, local
  databases, credentials, raw logs, workspace backups, delivery history, review
  history, or private infrastructure details.
- Route security issues through the private process in [SECURITY.md](SECURITY.md),
  not through public issues or pull requests.

## Support and Compatibility

Before `1.0`, compatibility is best effort and may change between implementation
milestones. OpenStudy does not currently promise a hosted service, remote HTTP
API, MCP server, published release, or automation runtime.
