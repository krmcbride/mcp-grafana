# GitHub Actions CI Best Practices

## Workflow Trigger Patterns

**Current Setup (Recommended):**

```yaml
on:
  push:
    branches: [master, main]
  pull_request:
    branches: [master, main]
```

**Benefits:**

- **PR Validation**: Catches issues before merge via required status checks
- **Post-merge Verification**: Ensures main branch health after merge
- **Branch Protection**: Enables "Require status checks to pass before merging"

**Alternative Patterns:**

- **PR-only**: `pull_request` only - saves CI resources but no post-merge validation
- **Push-only**: `push` only - validates main branch but not PRs before merge
- **Scheduled**: Add `schedule: cron: '0 2 * * *'` for nightly builds

## Matrix Build Strategy

**Current Cross-Platform Matrix:**

```yaml
strategy:
  matrix:
    os: [linux, darwin, windows]
    arch: [amd64, arm64]
    exclude:
      - os: windows
        arch: arm64
```

**Rationale:**

- **linux/amd64**: Primary development and CI platform
- **darwin/amd64+arm64**: macOS Intel and Apple Silicon support
- **windows/amd64**: Windows compatibility
- **Exclude windows/arm64**: Uncommon deployment target

## Job Dependencies and Gates

**ci-success Job Pattern:**

- Use `needs: [test, lint, build]` to wait for all required jobs
- Use `if: always()` to run even if some jobs fail
- Check individual job results with `needs.jobname.result`
- Provides single status check for branch protection rules

**Best Practices:**

- **Fail Fast**: Set `fail-fast: false` in matrix to see all platform failures
- **Caching**: Use `cache: true` in `setup-go` for faster dependency downloads
- **Permissions**: Use minimal `contents: read` permissions
- **Timeouts**: Add `timeout-minutes: 10` to prevent stuck jobs

## CI Target Usage

**Integration with Makefile:**

- `make test-ci`: Runs tests with coverage output (overrides vendor mode)
- `make format-ci`: Checks formatting without modifying files
- `make lint-ci`: Runs linters (overrides vendor mode)
- `make build-ci`: Cross-platform build validation (overrides vendor mode)

**Environment Variables:**

- `GOOS` and `GOARCH`: Set by matrix for cross-compilation
- `GOFLAGS=""`: CI targets override vendor mode to leverage Go module cache
- Local development still uses `GOFLAGS="-mod=vendor"` for offline builds

## Release Process

**Manual Release Workflow** (`.github/workflows/release.yml`)

- **Trigger**: Manual workflow dispatch with semantic version tag input
- **Pre-release validation**: `make format-ci`, `make test-ci`, `make lint-ci`
- **Tag creation**: Automated git tag creation and push
- **Release automation**: GoReleaser handles cross-platform builds and GitHub release

**Release Checklist:**

1. **Update CHANGELOG.md**:

   - Move items from `[Unreleased]` to new version section
   - Add release date: `## [1.0.0] - 2025-01-15`
   - Create new empty `[Unreleased]` section

2. **Trigger Release**:

   - GitHub Actions → "Release" workflow → "Run workflow"
   - Enter semantic version tag (e.g., `v1.0.0`)

3. **Post-Release**:
   - Verify GitHub release created with binaries
   - Update any external documentation if needed

**IMPORTANT**: Always maintain `CHANGELOG.md` manually - GoReleaser only generates GitHub release notes from commits.
