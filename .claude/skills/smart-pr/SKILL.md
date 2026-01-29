---
name: smart-pr
description: Generate or update PR title and description based on branch diffs
allowed-tools:
  - Bash(git branch*)
  - Bash(git diff*)
  - Bash(git log*)
  - Bash(gh pr view*)
  - Bash(gh pr create*)
  - Bash(gh pr edit*)
  - Bash(gh pr list*)
  - Bash(gh api*)
  - Read
  - Grep
  - Glob
---

# Smart PR Generator

This skill automatically generates or updates Pull Request titles and descriptions based on branch diffs. It analyzes code changes and uses AI to create meaningful, well-structured PR content.

## Prerequisites

- **GitHub CLI**: Must be authenticated via `gh auth login`
- **Git Repository**: Must be in a git repository
- **Not on Main Branch**: Must be on a feature/bug/task branch (not main/master)

## Usage

When invoked, this skill will:
1. Verify you're on a feature branch (not main/master)
2. Check if a PR already exists for the current branch
3. Analyze all changes between your branch and the base branch
4. Generate a PR title following conventional commit format
5. Generate a structured PR description with summary, changes, and testing checklist
6. Determine the appropriate PR label based on change type
7. Create a new PR or update the existing PR with generated content and label

## How it Works

### Step 1: Verify Git Context
- Check current branch: `git branch --show-current`
- Ensure not on main/master branch (error if on main)
- Determine base branch (try main, fall back to master)

### Step 2: Check for Existing PR
- Check if PR exists: `gh pr view --json number,title,body`
- Capture PR number if found

### Step 3: Gather Diff Information
- Get changes since branching: `git diff main...HEAD` (or `master...HEAD`)
- Get commit history: `git log main...HEAD --oneline`
- For large diffs (>5000 lines), use `git diff --stat` instead

### Step 4: Analyze Changes
Analyze the diff to understand:
- Which files changed (focus on: api/, eventgenerator/, scalingengine/, etc.)
- Nature of changes (feature, fix, refactor, docs, tests)
- Scope (single component vs multiple)
- Commit messages for context

### Step 5: Generate PR Title
**Format**: `<type>: <concise description>`

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring without behavior change
- `docs`: Documentation changes
- `test`: Test additions or modifications
- `chore`: Maintenance tasks (dependencies, configs, etc.)
- `perf`: Performance improvements

**Rules**:
- Maximum 72 characters
- Use imperative mood ("add" not "adds" or "added")
- Be specific and concise
- Focus on what, not how

**Examples**:
- `feat: add autoscaling for CPU metrics`
- `fix: resolve race condition in scaling engine`
- `refactor: simplify event generator aggregation logic`
- `docs: update deployment instructions for MTA`
- `test: add integration tests for scheduler component`

### Step 6: Generate PR Description
**Structure**:
```markdown
## Summary
- High-level bullet point 1 (what problem does this solve?)
- High-level bullet point 2 (what is the approach?)
- High-level bullet point 3 (any important context?)

## Changes
- Key technical change 1 (with file references if relevant)
- Key technical change 2
- Key technical change 3

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated (if applicable)
- [ ] Manual testing completed
- [ ] Documentation updated (if applicable)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

**Description Rules**:
- Focus on "why" not just "what"
- Reference key files/components (e.g., "Updated api/cmd/api/main.go")
- Include testing guidance
- Keep it scannable with bullets and clear sections
- Mention breaking changes or migration steps
- Reference relevant components: API Server, Event Generator, Scaling Engine, Scheduler (Java), Metrics Forwarder, Operator

### Step 7: Determine PR Label

This project requires exactly **one** label for release notes:
- `enhancement`: New features or improvements
- `bug`: Bug fixes
- `chore`: Maintenance (configs, dependencies, tooling, docs)
- `breaking-change`: Breaking backward compatibility
- `dependencies`: Dependency updates (only if primary purpose)

**Selection Logic**:
1. Breaking backward compatibility → `breaking-change`
2. Primary purpose is dependency updates → `dependencies`
3. Fixing a bug → `bug`
4. Adding feature or improving functionality → `enhancement`
5. Documentation, configs, tooling, maintenance → `chore`

**Examples**:
- New skill added → `enhancement`
- Fix race condition → `bug`
- Update CLAUDE.md documentation → `chore`
- Add Claude Code config → `chore`
- Upgrade Go version → `dependencies` (if that's the main change)
- Bump dependency versions only → `dependencies`
- Change API interface incompatibly → `breaking-change`

### Step 8: Create or Update PR
**If PR exists**:
- Update: `gh pr edit <number> --title "..." --body "..."`
- Add label: `gh pr edit <number> --add-label "<label>"`

**If no PR exists**:
- Create: `gh pr create --title "..." --body "..." --base main --label "<label>"`

**Use heredoc for multi-line body**:
```bash
gh pr create --title "feat: example" --body "$(cat <<'EOF'
## Summary
- Point 1

## Changes
- Change 1

🤖 Generated with Claude Code
EOF
)" --label "enhancement"
```

Display PR URL, title, and label after creation/update.

## Error Handling

**On Main Branch**:
- Error: "Cannot create PR from main/master branch. Please switch to a feature branch first."
- Suggest: `git checkout -b feature/my-new-feature`

**No Changes**:
- Warn: "No changes detected between your branch and main. Nothing to create a PR for."

**GitHub CLI Not Authenticated**:
- Error: "GitHub CLI not authenticated. Please run: gh auth login"

**Base Branch Doesn't Exist**:
- Try main first, then master
- If neither exists, error: "Could not find base branch (main or master)"

**Large Diff**:
- If diff >5000 lines, use `git diff --stat` instead of full diff
- Still generate meaningful title/description from file summary and commit messages

## Output

Display after successful execution:
- PR number (new or updated)
- Generated title
- Applied label
- Brief summary of changes
- Link to PR

Example:
```
✓ Updated PR #123 with generated content

Title: feat: add CPU-based autoscaling to event generator
Label: enhancement

Summary:
- Analyzed 15 files changed across eventgenerator and scalingengine components
- Detected new feature implementation with tests
- Generated description with 3 key changes and testing checklist

View PR: https://github.com/cloudfoundry/app-autoscaler/pull/123
```

## Tips

- Write descriptive commit messages to improve generation
- Push changes before running the skill
- Review and edit the PR after generation
- Add context in commit messages for complex changes

## Limitations

- Generates from code diffs and commits, not intent
- Very large PRs (100+ files) get more generic descriptions
- Does not auto-fill custom PR template fields
- Works best with focused PRs that have clear, single purposes
