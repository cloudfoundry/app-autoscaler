---
name: analyze-ci
description: Analyze failed GitHub Action jobs for the current branch's PR.
allowed-tools:
  - Bash(gh pr view*)
  - Bash(gh pr checks*)
  - Bash(gh run view*)
  - Bash(gh api*)
---

# Analyze CI Failures

This skill analyzes logs from failed GitHub Action jobs for the current branch's PR.

## Prerequisites

- **GitHub CLI**: Must be authenticated via `gh auth login`
- **Current Branch**: Must have an open PR on GitHub

## Usage

When invoked, this skill will:
1. Detect the PR number for the current branch
2. Check the status of all CI checks for that PR
3. Identify any failed jobs
4. Fetch and analyze the logs from failed jobs
5. Provide a concise summary with root causes and relevant error snippets

## How it works

The skill uses the GitHub CLI to:
- Detect current PR: `gh pr view --json number`
- List all checks for the PR: `gh pr checks <pr-number>`
- View detailed run information: `gh run view <run-id>`
- Fetch logs from failed jobs: `gh api repos/.../actions/jobs/<job-id>/logs`

Output includes:
- PR number and branch being analyzed
- Failed job names
- Root cause analysis
- Error messages and stack traces
- Relevant log snippets
- Suggested fixes (when applicable)
