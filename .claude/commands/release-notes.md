---
allowed-tools: Bash(git:*), Bash(gh:*)
description: Generate release notes from commits and update PR description
---

# Release Notes Generator

Generate release notes for the current branch and update the GitHub PR description.

## Steps

1. Get the current branch: `git branch --show-current`
2. Get commits since main: `git log origin/main..HEAD --oneline --no-merges`
3. Get diff summary: `git diff origin/main --stat`
4. Check if PR exists: `gh pr view --json number,title,body`

## Release Notes Format

Generate release notes with these sections (omit empty sections):

### Summary
One-sentence description of this release.

### What's New
- User-facing feature descriptions (from feat: commits)

### Bug Fixes
- Bug fix descriptions (from fix: commits)

### Breaking Changes
- Any breaking changes (from BREAKING CHANGE: or feat!: commits)

## After Generating

1. Show the generated release notes to the user
2. Ask: "Update the PR description with these release notes?"
3. If confirmed, prepend to existing PR body:
   `gh pr edit --body "[release notes]\n\n---\n\n[existing body]"`
