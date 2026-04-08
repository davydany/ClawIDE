# ClawIDE Documentation Screenshot Capture

Capture fresh screenshots of ClawIDE for documentation. This is a repeatable workflow for every release — it builds the binary, starts a clean instance with demo data, captures all screenshots, and restores the user's original state.

## Arguments

- `$ARGUMENTS` — (optional) space-separated flags:
  - `skip-build` — Skip `make all`, use the existing binary
  - `skip-restore` — Leave the fresh ClawIDE running after capture (don't restore backup)

## Prerequisites

- `jq` must be installed (`brew install jq`)
- Playwright must be installed in `docs-website/` (`cd docs-website && npm install && npx playwright install chromium`)
- Demo projects must exist at `~/projects/workspaces/` (csharp-api, golang-microservice, java-spring, nodejs-express, php-laravel, python-api, ruby-rails, rustapi)

## Workflow

Execute each step sequentially. If any step fails, jump to the Restore step to clean up before reporting the error.

### Step 1: Build (skip if $ARGUMENTS contains "skip-build")

1. `cd /Users/davydany/projects/ccmux`
2. Run `make all`
3. Verify `./clawide` binary exists and is freshly built

### Step 2: Backup ~/.clawide

1. Generate backup path: `BACKUP_DIR="/tmp/clawide-backup-$(date +%s)"`
2. If `~/.clawide` exists, run `mv ~/.clawide "$BACKUP_DIR"`
3. Print the backup location — you will need it for restore

### Step 3: Start fresh ClawIDE

1. Run in background: `./clawide --restart --projects-dir /Users/davydany/projects/workspaces &`
2. Poll until ready: loop `curl -sf http://localhost:9800/ > /dev/null` every 1 second, max 15 attempts
3. If it doesn't come up after 15 seconds, restore the backup and abort with an error

### Step 4: Seed demo data

1. Run `bash scripts/setup-screenshot-env.sh`
2. If the script fails, report which step failed, restore backup, and abort
3. Verify: read `~/.clawide/state.json` with jq to confirm 5 projects exist

### Step 5: Capture screenshots

1. `cd docs-website`
2. Run `node scripts/capture-screenshots.js`
3. Count the screenshots: `ls static/images/screenshots/*.png | wc -l`
4. Check for any suspiciously small files (<5KB): `find static/images/screenshots -name '*.png' -size -5k`
5. Report the count and any issues

### Step 6: Cleanup and restore

1. Kill the clawide process: read PID from `~/.clawide/clawide.pid` and `kill` it, or use `pkill -f "clawide.*projects-dir"`
2. Clean up git artifacts from workspace projects:
   - `rm -rf ~/projects/workspaces/python-api/.git`
   - Remove any `.worktrees` directories: `rm -rf ~/projects/workspaces/python-api/.worktrees`
3. Remove the temporary clawide state: `rm -rf ~/.clawide`
4. Restore the backup: `mv "$BACKUP_DIR" ~/.clawide` (skip if $ARGUMENTS contains "skip-restore")

### Step 7: Report

1. List all screenshots with file sizes: `ls -lh docs-website/static/images/screenshots/*.png`
2. Count total screenshots captured
3. Report completion status

## Error Recovery

If ANY step fails:
1. Always attempt the restore step (Step 6) before stopping
2. Report which step failed and why
3. The user's `~/.clawide` must be restored — losing their state is not acceptable

## Notes

- The onboarding welcome screen screenshot is captured BEFORE completing onboarding (the script handles this)
- Feature workspaces require a git repo, so the setup script does `git init` on `python-api`
- Modal screenshots (Skills Manager, Agents Manager, MCP Servers Manager) require a full page reload between each capture — Escape/close buttons are unreliable in headless Playwright
- The workspace projects at `~/projects/workspaces/` are NOT git repos by default — only `python-api` gets `git init` for feature workspace support
