# OpenStudy

OpenStudy is a local-first AgentOps memorization runtime for agents. The
supported agent path is a small `openstudy` runner plus a single-file skill.

## Install

Tell your agent:

```text
Install OpenStudy from https://github.com/yazanabuashour/openstudy.
Complete both required steps before reporting success:
1. Install and verify the openstudy runner binary with `openstudy --version`.
2. Register the OpenStudy skill from skills/openstudy/SKILL.md using your native skill system.
```

For the latest release:

```bash
sh -c "$(curl -fsSL https://github.com/yazanabuashour/openstudy/releases/latest/download/install.sh)"
```

For a pinned release:

```bash
OPENSTUDY_VERSION=v0.1.0 sh -c "$(curl -fsSL https://github.com/yazanabuashour/openstudy/releases/download/v0.1.0/install.sh)"
```

A complete install has two parts:

- `openstudy --version` succeeds
- the matching skill is registered from `skills/openstudy/SKILL.md`,
  `https://github.com/yazanabuashour/openstudy/tree/<tag>/skills/openstudy`,
  or `openstudy_<version>_skill.tar.gz`

Use the agent's native skill manager. OpenStudy does not require a specific
skill path or agent implementation.

## Upgrade

Tell your agent:

```text
Upgrade OpenStudy from https://github.com/yazanabuashour/openstudy.
Complete both required steps before reporting success:
1. Upgrade and verify the openstudy runner binary with `openstudy --version`.
2. Re-register the OpenStudy skill from skills/openstudy/SKILL.md using your native skill system.
```

Or upgrade the runner manually:

```bash
sh -c "$(curl -fsSL https://github.com/yazanabuashour/openstudy/releases/latest/download/install.sh)"
```

Then verify the runner and re-register the matching skill:

```bash
command -v openstudy
openstudy --version

```

## AgentOps Architecture

OpenStudy's agent-facing path is the AgentOps pattern: the skill gives the
agent task policy, and the local runner performs stateful memorization
operations through structured JSON. This keeps practice rules close to the
agent, avoids broad repo search and ad hoc human CLI flows, and leaves storage
local instead of requiring a hosted service.

OpenStudy treats this runner/skill architecture as its supported interface for
agents compared with traditional MCP or CLI-only integrations. The production
eval gate exercises the installed runner and skill only.

## Runner Interface

The skill sends structured JSON on stdin and reads structured JSON from stdout
for these runner domains:

```bash
openstudy cards
openstudy review
openstudy sources
openstudy windows
```

## Direct Go Package

OpenStudy `0.1.0` does not ship a supported public Go package or SDK. Go source
in this repository is intended for contributors and the released runner. Agent
installations should use the installed `openstudy` binary and registered skill.

## Local Storage

The default SQLite path is
`${XDG_DATA_HOME:-~/.local/share}/openstudy/openstudy.sqlite`. Override it with:

- `OPENSTUDY_DATABASE_PATH`
- `openstudy <domain> --db path`

The SQLite database is an implementation detail of the runner. Routine agent
work should use the installed runner, not direct SQLite access.

## Eval Evidence

The production runner/skill passed the 8-scenario `v0.1.0` release gate:
[`docs/evals/results/os7nh-v0.1.0.md`](docs/evals/results/os7nh-v0.1.0.md).

The eval protocol is documented in
[`docs/evals/agent-production.md`](docs/evals/agent-production.md).

## Development

Use the pinned local toolchain for repository development:

```bash
mise install
printf '%s\n' '{"action":"list_cards","status":"active","limit":10}' | \
  OPENSTUDY_DATABASE_PATH="$(mktemp -d)/openstudy.sqlite" mise exec -- go run ./cmd/openstudy cards
mise exec -- go test ./...
mise exec -- ./scripts/validate-agent-skill.sh
mise exec -- ./scripts/validate-committed-artifacts.sh
mise exec -- ./scripts/validate-release-docs.sh
```

`mise.toml` is the canonical local toolchain source; use `mise exec -- ...` for
repo checks.

## Releases

Tagged `v0.y.z` releases publish platform binary archives, the skill archive,
the installer, source archive, SHA256 checksums, a CycloneDX SBOM, and GitHub
attestations. Published release assets are intended to be immutable going
forward. See
[`docs/release-verification.md`](docs/release-verification.md) for verification
steps.

## Contributing

Outside contributors can work entirely through GitHub issues and pull requests.
Beads is maintainer-only workflow tooling and is not required for community
contributions.

See `CONTRIBUTING.md` for contribution expectations, `CODE_OF_CONDUCT.md` for
community standards, and `SECURITY.md` for vulnerability reporting.
