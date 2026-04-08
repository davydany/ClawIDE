# Generate Release Documentation

Analyze commits since the last release, identify new features, generate a changelog entry, and create or update feature documentation pages for each new feature.

## Arguments

- `$ARGUMENTS` — (optional) an explicit version string like `v1.2.0`. If omitted, the version is determined from `git describe`.

## Context

- Docs site is at `docs-website/` (Hugo, built with `npm run build`)
- Changelog lives at `docs-website/content/changelog.md` (Keep a Changelog format)
- Feature docs live at `docs-website/content/features/*.md` (Hugo markdown with frontmatter)
- Features index is at `docs-website/content/features/_index.md`
- Screenshots are at `docs-website/static/images/screenshots/`

## Workflow

### Step 1: Determine version boundaries

1. Get the latest release tag: `git tag -l --sort=-v:refname | head -1`
2. If `$ARGUMENTS` provides a version, use that as the new version. Otherwise derive it from `git describe --tags`.
3. Store: `LAST_TAG` (e.g., `v1.0.0`) and `NEW_VERSION` (e.g., `v1.1.0`)
4. Print: "Generating docs for $NEW_VERSION (commits since $LAST_TAG)"

### Step 2: Analyze commits

1. Run `git log --oneline $LAST_TAG..HEAD` to get all commits since the last release.
2. Run `git log --stat $LAST_TAG..HEAD` to understand the scope of changes (which files were touched).
3. Run `git diff --stat $LAST_TAG..HEAD` to see the overall diff summary.
4. Count the commits: `git rev-list --count $LAST_TAG..HEAD`

### Step 3: Identify features

For each commit, classify it as one of:
- **Feature** (new capability) — commit messages with "add", "implement", "introduce", "support", "new"
- **Change** (modification to existing behavior) — commit messages with "change", "update", "improve", "refactor", "replace"
- **Fix** (bug fix) — commit messages with "fix", "resolve", "correct", "patch"

For each identified feature:
1. Read the actual code changes: `git show <commit-hash> --stat` to find which files were modified
2. Read the relevant handler files, templates, and JS files to understand what the feature does from a user's perspective
3. Check if a feature doc already exists in `docs-website/content/features/`

Build a structured list of features with:
- Name (human-readable, bolded)
- Category (Added / Changed / Fixed)
- One-line summary
- Detailed description of what it does and how users interact with it
- Key files involved (for reference, not for the docs)
- Whether it needs a new docs page or an update to an existing one

Present this list to the user with AskUserQuestion before proceeding. Ask:
- "Here are the features I identified for $NEW_VERSION. Should I proceed with generating docs for all of them, or would you like to add/remove/modify any?"

### Step 4: Generate changelog entry

1. Read `docs-website/content/changelog.md` to understand the existing format.
2. Generate a new version section at the top (below the frontmatter and intro paragraph), following the Keep a Changelog format:

```markdown
## [X.Y.Z] — YYYY-MM-DD

Brief 1-2 sentence summary of the release theme. This release includes N commits since $LAST_TAG.

### Added
- **Feature Name**: Description of what it does and why it matters. Include key capabilities.

### Changed
- **Change Name**: What changed and how it differs from the previous behavior.

### Fixed
- **Fix Name**: What was broken and how it's now corrected.
```

Rules for the changelog:
- Each entry starts with a bold feature name followed by a colon
- Descriptions are 1-2 sentences — enough to understand the feature without reading the full docs
- Group by Added/Changed/Fixed (omit empty sections)
- Include the commit count in the intro paragraph
- Date format: YYYY-MM-DD
- Link-worthy features should mention their docs page implicitly through the description

### Step 5: Create or update feature documentation pages

For each **new feature** that warrants its own page (not minor fixes or small changes):

1. Determine if a page already exists in `docs-website/content/features/`
2. If it exists, update it with new information from the code changes
3. If it doesn't exist, create a new page following this template:

```markdown
---
title: "Feature Name"
description: "One-line description for SEO and page metadata."
weight: NN
---

Introductory paragraph explaining what the feature does and why it's useful. 2-3 sentences max.

{{< screenshot src="feature-name.png" alt="ClawIDE Feature Name" caption="Brief caption describing what the screenshot shows" >}}

## Section 1 (e.g., "How It Works", "Getting Started")

Explain the core workflow or concept.

## Section 2 (e.g., "Creating a X", "Configuration")

Step-by-step instructions:

1. Do this.
2. Then do that.
3. Result.

## Section 3 (e.g., "Advanced Options", "API")

Additional details, tables, or reference information.
```

Guidelines for feature docs:
- **Read the actual implementation** (handlers, templates, JS) before writing — don't guess at behavior
- **Weight**: Use the next available number after existing pages (check `docs-website/content/features/*.md` for current weights)
- **Screenshot reference**: Use `{{< screenshot src="feature-name.png" >}}` — the screenshot may or may not exist yet (it will be captured by `/release-screenshots`)
- **Cross-references**: Link to related features using `{{< ref "features/other-feature" >}}`
- **Code examples**: Use fenced code blocks for CLI commands, config files, API payloads
- **Keep it practical**: Focus on how to use the feature, not how it's implemented internally
- **No fluff**: Every sentence should teach the user something. No "In this section we will..."

### Step 6: Update the features index

1. Read `docs-website/content/features/_index.md`
2. For each new feature page created in Step 5, add an entry to the index in the appropriate position:

```markdown
## Feature Name

Brief description matching the feature doc's intro paragraph.

[Learn more →]({{< ref "features/feature-name" >}})
```

3. Place new entries logically — group similar features near each other, not just appended at the bottom.

### Step 7: Verify the docs build

1. Run `cd docs-website && npx hugo --minify --gc 2>&1`
2. Check for build errors — Hugo will fail if there are broken `ref` links or malformed frontmatter
3. Report the page count and any warnings

### Step 8: Report

Present a summary to the user:
- Version documented
- Number of commits analyzed
- Changelog entry added (show the first few lines)
- Feature pages created/updated (list filenames)
- Features index updated (yes/no)
- Hugo build status (pass/fail with page count)
- Reminder to run `/release-screenshots` to capture screenshots for any new feature pages

## Important Notes

- **Do NOT fabricate features.** Every changelog entry must trace back to a real commit. If a commit message is ambiguous, read the actual code diff to understand what changed.
- **Do NOT duplicate existing docs.** If a feature was documented in a previous release, update the existing page rather than creating a new one.
- **Read code before writing docs.** For each feature, read at least the handler/template/JS file to understand the exact behavior, API format, and UI flow. Don't write docs based solely on commit messages.
- **Respect existing format.** Match the tone, depth, and structure of existing feature pages in `docs-website/content/features/`. Read 2-3 existing pages before writing new ones.
- **Don't touch screenshots.** This skill only generates text content. Screenshots are handled by `/release-screenshots`.
