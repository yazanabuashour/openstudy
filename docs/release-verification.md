# Release Verification

OpenStudy releases are tagged GitHub Releases with downloadable runner, skill,
source, checksum, SBOM, and installer assets. Maintainers can build the same
candidate bundle locally with:

```bash
mise exec -- ./scripts/build-release-bundle.sh <version> <out-dir>
```

The generated artifact set is:

- `openstudy_<version>_<os>_<arch>.tar.gz`
- `openstudy_<version>_skill.tar.gz`
- `openstudy_<version>_source.tar.gz`
- `openstudy_<version>_checksums.txt`
- `openstudy_<version>_sbom.json`
- `install.sh`

Platform archives contain the production `openstudy` binary. The skill archive
contains `skills/openstudy/SKILL.md`. The source archive is the canonical Go
module source artifact with maintainer-only local metadata excluded. Checksums
cover every generated archive, the SBOM, and the install script.

Before publication, run the release-blocking production eval:

```bash
mise exec -- go run ./scripts/agent-eval/os7nh run
```

The eval model pin is `gpt-5.4-mini`. A release candidate must not publish if
the production gate fails, if the skill bypass policy fails, or if committed
reports include raw logs, direct SQLite workflows, HTTP, MCP, source-built
runner paths, unsupported transports, machine-local paths, or maintainer-only
reference details.

## Verify A Bundle

From the generated bundle directory:

```bash
shasum -a 256 -c openstudy_<version>_checksums.txt
tar -tzf openstudy_<version>_skill.tar.gz
tar -tzf openstudy_<version>_source.tar.gz
```

When artifacts are attached to a hosted release, verify attestations for the
platform archive, skill archive, source archive, SBOM, and installer before
publishing:

```bash
gh attestation verify openstudy_<version>_<os>_<arch>.tar.gz --repo yazanabuashour/openstudy
gh attestation verify openstudy_<version>_skill.tar.gz --repo yazanabuashour/openstudy
gh attestation verify openstudy_<version>_source.tar.gz --repo yazanabuashour/openstudy
gh attestation verify openstudy_<version>_sbom.json --repo yazanabuashour/openstudy
gh attestation verify install.sh --repo yazanabuashour/openstudy
```

## Smoke-Test An Install

Install into a temporary directory, then verify the runner version and domains:

```bash
install_dir="$(mktemp -d)"
OPENSTUDY_INSTALL_DIR="$install_dir" \
  OPENSTUDY_VERSION=v0.1.0 \
  sh ./scripts/install.sh

export PATH="$install_dir:$PATH"
command -v openstudy
openstudy --version
openstudy --help
```

The valid runner domains are `cards`, `review`, `sources`, and `windows`.

## Verify Release Notes

Each tag must have matching release notes and changelog coverage:

```bash
mise exec -- ./scripts/validate-release-docs.sh v0.1.0
```

Release notes must link the reduced production eval report, summarize artifact
verification, and avoid raw logs, local databases, machine-local paths, and
private material.

## Immutability

Published assets are immutable. If a released artifact is wrong, ship a new
patch release instead of replacing the existing tag or assets. Direct SQLite,
HTTP, MCP, source-built runner, and ad hoc script workflows are not alternate
release verification paths.
