# ClawIDE Release Workflow

Perform a versioned release of ClawIDE: commit, push, wait for CI, tag, and verify the full release pipeline.

## Arguments

- `$ARGUMENTS` — (optional) version bump type: `patch` (default), `minor`, or `major`. You can also pass an explicit version like `v0.2.0`.

## Workflow

### Step 1: Pre-flight checks

1. Run `git status` and `git diff --stat` to see what's changed.
2. If there are no changes (working tree clean, nothing to commit), skip to Step 3 (tagging) — the user may just want to re-tag or bump the version on the current HEAD.
3. If there are uncommitted changes, proceed to Step 2.

### Step 2: Commit and push

1. Run `git diff` to review all changes in detail.
2. Run `git log --oneline -5` to see recent commit message style.
3. Stage only the relevant modified files by name — never use `git add -A` or `git add .`.
4. Write a concise commit message that describes the **why** of the changes. Follow the existing project style (imperative mood, no prefix tags).
5. Commit the changes.
6. Push to `origin master`.

### Step 3: Wait for CI

1. Run `gh run list --branch master --limit 1` to find the CI run triggered by the push.
2. Run `gh run watch <run-id> --exit-status` to wait for CI to complete.
3. If CI fails, inspect the logs with `gh run view <run-id> --log-failed`, diagnose the issue, and stop — do NOT tag a failing build. Report the failure to the user and suggest fixes.

### Step 4: Determine the next version

1. Get the latest tag: `git tag -l --sort=-v:refname | head -1`.
2. Parse the current version (e.g., `v0.1.2` -> major=0, minor=1, patch=2).
3. Apply the bump based on `$ARGUMENTS`:
   - `patch` (default): increment patch -> `v0.1.3`
   - `minor`: increment minor, reset patch -> `v0.2.0`
   - `major`: increment major, reset minor+patch -> `v1.0.0`
   - Explicit version (e.g., `v0.2.0`): use as-is after validating it's greater than current
4. Confirm the version with the user before proceeding using AskUserQuestion: "Release as `vX.Y.Z`?"

### Step 5: Tag and push

1. Create an annotated tag: `git tag -a vX.Y.Z -m "vX.Y.Z: <brief summary of changes since last tag>"`.
   - The summary should be derived from `git log --oneline <last-tag>..HEAD`.
2. Push the tag: `git push origin vX.Y.Z`.

### Step 6: Watch the release pipeline

1. Wait a few seconds, then find the Release workflow run: `gh run list --limit 5` and filter for the tag branch.
2. Watch it with `gh run watch <run-id> --exit-status`.
3. The pipeline has three jobs that must all pass:
   - **test** — Go vet + tests
   - **release** — Cross-compile (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) + checksums + GitHub Release
   - **docker** — Multi-arch Docker image pushed to `ghcr.io/davydany/clawide:<version>` and `:latest`

### Step 7: Verify the release

1. Run `gh release view vX.Y.Z` to confirm the release was created with all expected assets:
   - 4 tar.gz archives (darwin-amd64, darwin-arm64, linux-amd64, linux-arm64)
   - checksums.txt
2. Report the final summary to the user:
   - Version released
   - Asset count and sizes
   - Docker image tags
   - Link to the GitHub release page

### On failure at any step

- Do NOT skip steps or continue past failures.
- Report exactly what failed and at which step.
- If CI or the release pipeline fails, show the relevant log output and suggest a fix.
- Never force-push, delete tags, or take destructive recovery actions without explicit user approval.
