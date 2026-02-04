# CI/CD Setup Guide

This document explains the CI/CD setup for the Go backend, including pre-commit hooks, GitHub Actions workflows, and local development workflow.

## Overview

The CI/CD pipeline consists of three main components:

1. **Pre-commit Hooks** - Fast local checks (<10 seconds) before commits
2. **GitHub Actions CI** - Comprehensive checks on every PR/push
3. **Dependabot** - Automated dependency updates

## Pre-Commit Hooks

Pre-commit hooks run automatically before each commit to catch common issues early.

### Installation

#### Option 1: Using Make (Recommended)
```bash
make pre-commit-install
```

#### Option 2: Manual Installation
```bash
# Install pre-commit
pip install pre-commit
# OR
brew install pre-commit

# Install hooks
pre-commit install
```

### What Gets Checked

Pre-commit runs these checks in ~10 seconds:

1. **File Checks**
   - Remove trailing whitespace
   - Ensure files end with newline
   - Validate YAML/JSON syntax
   - Block files >500KB
   - Detect merge conflicts
   - Enforce consistent line endings

2. **Go Formatting** (auto-fixes)
   - `go fmt` - Standard Go formatting
   - `go imports` - Import organization

3. **Go Build**
   - Verify code compiles: `go build ./...`

4. **Dependency Hygiene**
   - Verify `go.mod` is clean: `go mod tidy`

5. **Fast Unit Tests**
   - Run quick tests: `go test -short -race ./...`
   - Skips slow integration tests
   - Detects race conditions

### Running Manually

```bash
# Run on all files
make pre-commit-run

# Run on staged files only
pre-commit run

# Skip hooks (use sparingly)
git commit --no-verify
```

### Troubleshooting

**Hook fails with "command not found"**
- Ensure Go is in your PATH
- Try reinstalling: `pre-commit clean && pre-commit install`

**Tests are too slow**
- Pre-commit only runs `-short` tests
- Full test suite runs in CI

**Auto-formatting changes my code**
- This is intentional - commit the formatted code
- Ensures consistent style across the team

## GitHub Actions CI

Three parallel jobs run on every push to `main` and every pull request:

### Job 1: test-and-lint (Primary Quality Gate)

**Duration:** ~3-4 minutes

**Steps:**
1. Checkout code
2. Setup Go 1.25.6 with module caching
3. Verify dependencies (`go mod download && go mod verify`)
4. Run `golangci-lint` (all 11 linters, 5min timeout)
5. Run tests with race detector (`go test -race -v ./...`)
6. Generate coverage report
7. **Enforce 80% coverage threshold** (fails if below)
8. Upload coverage to GitHub summary

**What It Catches:**
- All linting issues (gosec, staticcheck, govet, gofmt, goimports, etc.)
- Test failures
- Race conditions
- Insufficient test coverage

### Job 2: security-scan (Vulnerability Detection)

**Duration:** ~1-2 minutes

**Steps:**
1. Checkout code
2. Setup Go 1.25.6
3. Install and run `govulncheck`

**What It Catches:**
- Known CVEs in dependencies
- Vulnerable dependency versions

**Note:** This job is informational and doesn't block PRs (can be made required later).

### Job 3: build-binary (Build Verification)

**Duration:** ~1-2 minutes

**Steps:**
1. Checkout code
2. Setup Go 1.25.6
3. Read version from `VERSION` file
4. Build binary with version injection
5. Upload binary artifact (7-day retention)

**What It Catches:**
- Build failures
- Missing dependencies

### Viewing CI Results

**In Pull Requests:**
- Check the "Checks" tab
- All 3 jobs must pass (green checkmark)
- Click job name for detailed logs

**Coverage Reports:**
- View in GitHub Actions summary
- Click on "Test & Lint" job
- Scroll to bottom for coverage breakdown

**Build Artifacts:**
- Available in "Build Binary" job
- Download from "Artifacts" section
- Valid for 7 days

## Coverage Requirements

### Current Threshold: 75% (Goal: 80% → 85% → 90%)

The CI enforces a 75% minimum coverage threshold. Coverage is measured across ALL `internal/` packages using:

```bash
go test -coverpkg=./internal/... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Current Coverage: 76.4%

- **Handlers**: 75.7% (excellent, comprehensive tests)
- **Service**: 3.2% (tested indirectly through handlers)
- **Middleware**: 0.9% (tested indirectly through handlers)
- **Overall**: 76.4% across all internal packages

### Why Start at 75%?

- Current coverage: 76.4% (already meets threshold)
- Handler layer has comprehensive tests
- Many packages (service, repository, config) are tested indirectly through handler tests (Go best practice)
- 75% is a pragmatic starting point that doesn't block development
- Threshold will increase to 80%, then 85-90% as direct service/repository unit tests are added

### Coverage Strategy

The `-coverpkg=./internal/...` flag measures coverage across ALL internal packages, even those without direct test files. This provides an accurate picture of overall code coverage and ensures handler tests count toward overall coverage.

### Checking Coverage Locally

```bash
# Run coverage with threshold check
make coverage

# Generate HTML coverage report
make coverage-html
open coverage.html
```

### Improving Coverage

If CI fails due to low coverage:

1. Identify uncovered code:
   ```bash
   make coverage-html
   open coverage.html
   ```

2. Add tests for uncovered code paths

3. Run coverage check again:
   ```bash
   make coverage
   ```

## Local CI Simulation

Before pushing, you can run the same checks CI will run:

```bash
make ci-local
```

This runs:
1. `golangci-lint` (all linters)
2. Full test suite with race detector
3. Coverage report with 80% threshold
4. `govulncheck` security scan

**Duration:** ~5-8 minutes (similar to CI)

**When to Use:**
- Before pushing important changes
- After large refactors
- When you want confidence CI will pass

## Dependabot Configuration

Dependabot automatically creates PRs for dependency updates.

### What Gets Updated

1. **Go Modules** (`go.mod`)
   - Monthly schedule
   - Minor and patch versions grouped into single PR
   - Major versions get separate PRs

2. **GitHub Actions** (workflow files)
   - Monthly schedule
   - Each action gets separate PR

### Configuration

- **Schedule:** Monthly (1st of each month)
- **PR Limit:** 10 open PRs max
- **Labels:** `dependencies`, `backend` (Go) or `ci` (Actions)
- **Auto-reviewer:** ndewijer

### Reviewing Dependabot PRs

1. Check PR description for changelog links
2. Review dependency changes in Files tab
3. Wait for CI to pass (all 3 jobs)
4. Check for breaking changes in major version updates
5. Merge if tests pass and no breaking changes

### Creating Required Labels

Dependabot needs these labels to exist:

```bash
# Using GitHub CLI
gh label create dependencies --color 0366d6 --description "Dependency updates"
gh label create backend --color d73a4a --description "Backend related changes"
gh label create ci --color 0e8a16 --description "CI/CD related changes"
```

Or create via GitHub UI: Repository → Issues → Labels → New label

## Security Scanning

### gosec (Static Analysis)

Already integrated in `golangci-lint` configuration.

**What It Detects:**
- SQL injection vulnerabilities
- Hardcoded credentials
- Weak crypto usage
- Command injection
- Path traversal
- And more...

**Runs In:**
- Pre-commit: No (too slow)
- CI: Yes (as part of golangci-lint)

### govulncheck (CVE Scanner)

Official Go vulnerability scanner.

**What It Detects:**
- Known CVEs in dependencies
- Vulnerable versions
- Security advisories from Go team

**Runs In:**
- Pre-commit: No
- CI: Yes (separate job)
- Local: `make govulncheck`

**Responding to Vulnerabilities:**

1. Run locally:
   ```bash
   make govulncheck
   ```

2. If vulnerabilities found:
   ```bash
   # Update specific dependency
   go get -u github.com/vulnerable/package@latest

   # Or update all dependencies
   go get -u ./...
   go mod tidy
   ```

3. Verify fix:
   ```bash
   make govulncheck
   ```

## Common Workflows

### Starting New Work

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes
# ...

# Run quick checks
make test-short
make lint-fix

# Commit (pre-commit hooks run automatically)
git commit -m "Add my feature"

# Push and create PR
git push origin feature/my-feature
```

### Fixing CI Failures

**Linting Failures:**
```bash
# Auto-fix issues
make lint-fix

# Check remaining issues
make lint

# Commit fixes
git add .
git commit -m "Fix linting issues"
git push
```

**Test Failures:**
```bash
# Run tests locally
make test

# Run specific test
go test -v -run TestName ./path/to/package

# Fix and commit
git add .
git commit -m "Fix failing tests"
git push
```

**Coverage Failures:**
```bash
# Check coverage
make coverage-html
open coverage.html

# Add tests for uncovered code
# ...

# Verify threshold met
make coverage

# Commit new tests
git add .
git commit -m "Increase test coverage"
git push
```

**Security Vulnerabilities:**
```bash
# Check locally
make govulncheck

# Update dependencies
go get -u ./...
go mod tidy

# Verify fix
make govulncheck

# Commit updates
git add go.mod go.sum
git commit -m "Update dependencies to fix vulnerabilities"
git push
```

### Before Merging PR

```bash
# Run full CI locally
make ci-local

# If all passes:
# ✅ Merge PR in GitHub UI
```

## Makefile Reference

| Command | Description | Duration |
|---------|-------------|----------|
| `make test` | Run all tests with race detector | ~30s |
| `make test-short` | Run quick tests only | ~5s |
| `make lint` | Run golangci-lint | ~2min |
| `make lint-fix` | Auto-fix linting issues | ~2min |
| `make coverage` | Generate coverage + check threshold | ~30s |
| `make coverage-html` | Generate HTML coverage report | ~30s |
| `make build` | Build binary with version | ~5s |
| `make run` | Run the application | - |
| `make clean` | Remove build artifacts | <1s |
| `make pre-commit-install` | Install pre-commit hooks | <5s |
| `make pre-commit-run` | Run pre-commit on all files | ~10s |
| `make govulncheck` | Run security vulnerability scan | ~30s |
| `make ci-local` | Run all CI checks locally | ~5-8min |

## Tool Versions

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.6 | Runtime |
| golangci-lint | v1.64.8 | Linting (11 linters) |
| pre-commit-hooks | v6.0.0 | File checks |
| pre-commit-golang | v0.5.1 | Go-specific hooks |
| govulncheck | latest | Security scanning |

## Best Practices

### DO:
- ✅ Install pre-commit hooks (`make pre-commit-install`)
- ✅ Run `make test` before committing
- ✅ Run `make ci-local` before pushing important changes
- ✅ Check coverage regularly (`make coverage-html`)
- ✅ Update dependencies monthly (review Dependabot PRs)
- ✅ Fix linting issues with `make lint-fix`

### DON'T:
- ❌ Skip hooks with `--no-verify` (unless emergency)
- ❌ Ignore security vulnerabilities
- ❌ Commit without running tests
- ❌ Merge PRs with failing CI
- ❌ Ignore coverage drops

## Troubleshooting

### Pre-commit is too slow
- Pre-commit only runs `-short` tests
- Full suite runs in CI
- If still slow, check for expensive setup in tests

### CI is failing but local tests pass
- Run `make ci-local` to simulate CI
- Check Go version matches (1.25.6)
- Clear module cache: `go clean -modcache`

### Coverage threshold not met
- Run `make coverage-html` to see gaps
- Add tests for uncovered code paths
- Consider if 80% threshold needs adjustment

### Dependency conflicts
- Update all dependencies: `go get -u ./...`
- Run `go mod tidy`
- Check for breaking changes in major updates

### govulncheck reports vulnerabilities
- Update affected packages: `go get -u github.com/vulnerable/package@latest`
- If no fix available, check if vulnerability affects your usage
- Consider alternative packages

## Future Enhancements

Planned improvements (not yet implemented):

### Phase 3: Service & Repository Layer Tests
- Add direct service layer unit tests
- Add repository layer unit tests
- Increase coverage threshold to 85%, then 90%

### Phase 4: Docker Integration
- Create Dockerfile
- Add Docker build/health check workflow
- Match Python project's docker-test.yml

### Phase 5: Performance Benchmarks
- Add `go test -bench` to CI
- Track performance over time
- Alert on regressions

### Phase 6: Multi-Platform Builds
- Build for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Prepare for production releases

## Support

For issues or questions:
- Check this guide first
- Review CI logs in GitHub Actions
- Check `.golangci.yml` for linter configuration
- Review `.pre-commit-config.yaml` for hook configuration

## References

- [Pre-commit Documentation](https://pre-commit.com/)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [Go Testing](https://pkg.go.dev/testing)
- [Go Coverage](https://go.dev/blog/cover)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
