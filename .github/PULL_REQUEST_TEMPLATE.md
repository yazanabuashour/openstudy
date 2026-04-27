## Summary


## Verification

- [ ] `git diff --check`
- [ ] `mise exec -- go test ./...`
- [ ] `mise exec -- ./scripts/validate-agent-skill.sh`
- [ ] `mise exec -- ./scripts/validate-committed-artifacts.sh`
- [ ] `mise exec -- ./scripts/validate-release-docs.sh`

## Safety

- [ ] No private study material, local databases, raw logs, credentials, or machine-local paths are included.
- [ ] Public docs and examples are repo-relative or use neutral placeholders.
- [ ] Runner, skill, or release contract changes are documented.
