# /clawide-update

A Claude Code skill that automates the full ClawIDE release workflow — from committing code through verifying the deployed release artifacts.

## Location

```
.claude/commands/clawide-update.md
```

## Usage

```
/clawide-update              # Patch bump (v0.1.2 -> v0.1.3)
/clawide-update minor        # Minor bump (v0.1.2 -> v0.2.0)
/clawide-update major        # Major bump (v0.1.2 -> v1.0.0)
/clawide-update v0.3.0       # Explicit version
```

If no argument is provided, the skill defaults to a **patch** bump.

## What It Does

The skill runs a 7-step pipeline, stopping on any failure:

### 1. Pre-flight Checks

- Runs `git status` and `git diff --stat` to inspect the working tree.
- If the tree is clean, skips straight to tagging (useful for re-releasing or bumping without code changes).

### 2. Commit and Push

- Reviews all changes via `git diff`.
- Stages only the relevant files by name (never `git add -A`).
- Writes a descriptive commit message in imperative mood.
- Pushes to `origin master`.

### 3. CI Gate

- Finds the CI workflow run triggered by the push.
- Watches it until completion.
- If CI fails, the skill stops, shows the failure logs, and suggests fixes. **A failing build is never tagged.**

### 4. Version Bump

- Reads the latest tag (`git tag -l --sort=-v:refname`).
- Computes the next version based on the argument (patch/minor/major/explicit).
- Asks for confirmation before proceeding.

### 5. Tag and Push

- Creates an annotated tag with a brief summary derived from `git log --oneline <last-tag>..HEAD`.
- Pushes the tag to origin, which triggers the Release workflow.

### 6. Release Pipeline Watch

The Release workflow (`.github/workflows/release.yml`) runs three jobs sequentially:

| Job | What it does | Typical duration |
|-----|-------------|-----------------|
| **test** | `go vet` + `go test -race ./...` | ~15s |
| **release** | Cross-compile for 4 platforms, generate checksums, create GitHub Release | ~60s |
| **docker** | Build and push multi-arch Docker image to GHCR | ~5-6 min |

The skill watches all three jobs to completion.

### 7. Verification

Confirms the GitHub Release contains all expected artifacts:

- `clawide-<version>-darwin-amd64.tar.gz`
- `clawide-<version>-darwin-arm64.tar.gz`
- `clawide-<version>-linux-amd64.tar.gz`
- `clawide-<version>-linux-arm64.tar.gz`
- `checksums.txt`
- Docker images: `ghcr.io/davydany/clawide:<version>` and `ghcr.io/davydany/clawide:latest`

Reports a summary with asset sizes and the release URL.

## Failure Handling

- The skill never skips steps or continues past failures.
- CI or pipeline failures are reported with relevant log output.
- No destructive recovery actions (force-push, tag deletion) are taken without explicit approval.

## Prerequisites

- `gh` CLI authenticated with repo access
- `git` configured with push access to `origin`
- Working tree on the `master` branch

## Examples

**Typical patch release after a bug fix:**

```
> Made some changes to fix a bug...
> /clawide-update
# Commits, pushes, waits for CI, bumps v0.1.2 -> v0.1.3, tags, watches release, verifies
```

**Minor release for a new feature:**

```
> Finished implementing the new feature...
> /clawide-update minor
# Commits, pushes, waits for CI, bumps v0.1.3 -> v0.2.0, tags, watches release, verifies
```

**Tag-only release (no pending changes):**

```
> /clawide-update
# Detects clean tree, skips to version bump, tags HEAD, watches release, verifies
```
