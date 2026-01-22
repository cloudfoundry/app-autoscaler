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
3. Analyze all changes between your branch and the base branch (main/master)
4. Generate an intelligent PR title following conventional commit format
5. Generate a structured PR description with summary, changes, and testing checklist
6. Either create a new PR or update the existing PR with generated content

## How it Works

### Step 1: Verify Git Context
- Check current branch name using `git branch --show-current`
- Verify not on main/master branch (error if true)
- Determine base branch (try main, fallback to master)

### Step 2: Check for Existing PR
- Use `gh pr view --json number,title,body` to check if PR exists for current branch
- Capture PR number if found for later update

### Step 3: Gather Diff Information
- Run `git diff main...HEAD` (or `master...HEAD`) to get all changes since branching
- Run `git log main...HEAD --oneline` to see commit history for context
- For large diffs (>5000 lines), use `git diff --stat` for file summary instead

### Step 4: Analyze Changes
Analyze the diff content to understand:
- What files changed (focus on critical directories: api/, eventgenerator/, scalingengine/, etc.)
- Nature of changes (feature, bug fix, refactor, docs, tests)
- Scope of changes (single component vs multiple)
- Review commit messages for additional context

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
- Maximum 72 characters (GitHub best practice)
- Use imperative mood ("add" not "adds" or "added")
- Be specific but concise
- Focus on what the change accomplishes, not how

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
- Focus on "why" not just "what" - explain the rationale
- Reference key files/components changed (e.g., "Updated api/cmd/api/main.go")
- Include testing guidance specific to the changes
- Keep it scannable (bullets, clear sections)
- Mention any breaking changes or migration steps
- For this codebase, reference relevant components:
  - API Server, Event Generator, Scaling Engine, Scheduler (Java), Metrics Forwarder, Operator

### Step 7: Create or Update PR
- **If PR exists**: Use `gh pr edit <number> --title "..." --body "..."`
- **If no PR exists**: Use `gh pr create --title "..." --body "..." --base main`
- Use heredoc syntax for body to handle multi-line content properly:
  ```bash
  gh pr create --title "feat: example" --body "$(cat <<'EOF'
  ## Summary
  - Point 1

  ## Changes
  - Change 1

  🤖 Generated with Claude Code
  EOF
  )"
  ```
- Display the PR URL to user after creation/update

## Error Handling

Handle these scenarios gracefully:

1. **On Main Branch**:
   - Error message: "Cannot create PR from main/master branch. Please switch to a feature branch first."
   - Suggest: `git checkout -b feature/my-new-feature`

2. **No Changes**:
   - Warn: "No changes detected between your branch and main. Nothing to create a PR for."

3. **GitHub CLI Not Authenticated**:
   - Error message: "GitHub CLI not authenticated. Please run: gh auth login"

4. **Base Branch Doesn't Exist**:
   - Try main first, then master
   - If neither exists, error: "Could not find base branch (main or master)"

5. **Large Diff**:
   - If diff >5000 lines, use `git diff --stat` summary instead of full diff
   - Still generate meaningful title/description based on file summary and commit messages

## Output

After successful execution, display:
- PR number (new or updated)
- Generated title
- Brief summary of changes analyzed
- Direct link to PR on GitHub

Example output:
```
✓ Updated PR #123 with generated content

Title: feat: add CPU-based autoscaling to event generator

Summary:
- Analyzed 15 files changed across eventgenerator and scalingengine components
- Detected new feature implementation with corresponding tests
- Generated description with 3 key changes and testing checklist

View PR: https://github.com/cloudfoundry/app-autoscaler/pull/123
```

## Tips for Best Results

- Make descriptive commit messages - they help inform the PR title/description generation
- Push your changes before running the skill so the remote branch exists
- Review and manually edit the PR after generation if needed - this is a starting point
- For complex changes, consider adding more context in commit messages

## Limitations

- Skill generates based on code diffs and commits - it can't read your mind about intent
- For very large PRs (100+ files), description may be more generic
- Does not auto-fill custom PR template fields (but you can manually edit after)
- Works best with focused PRs that have clear, single purposes
