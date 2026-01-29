---
name: simplify
description: Simplify and refine code for clarity and maintainability
allowed-tools:
  - Task
---

# Code Simplifier

Simplifies and refines code for clarity, consistency, and maintainability while preserving all functionality.

## Usage

When invoked:
1. Launches the code-simplifier agent
2. Analyzes recently modified code (or specified code)
3. Applies refinements to improve clarity and maintainability
4. Preserves all functionality - only improves structure

## How it Works

Uses the Task tool to launch the **code-simplifier** agent with `subagent_type: "code-simplifier:code-simplifier"`. The agent:
- Focuses on recently modified code by default
- Applies project best practices from CLAUDE.md
- Reduces complexity and improves naming
- Never changes functionality

## Instructions

When invoked:

1. **Launch the code-simplifier agent** with Task tool:
   - `subagent_type: "code-simplifier:code-simplifier"`
   - `description: "Simplify and refine code"`
   - `prompt`: Provide context about what to simplify:
     - If user specified files/directories, include in prompt
     - If no scope, focus on recently modified code
     - Example: "Simplify the recently modified code, focusing on clarity and maintainability while preserving all functionality"

2. **Pass relevant context**:
   - Specific files or components to simplify
   - User concerns (e.g., "the reviewer said it's too complex")
   - Key project areas: api, eventgenerator, scalingengine, metricsforwarder, operator

3. **Let the agent work autonomously**. It will:
   - Identify target files
   - Read and analyze code
   - Apply simplifications using Edit tool

## Example Invocations

**Basic usage:**
```
User: /simplify
Claude: [Launches code-simplifier agent to simplify recently modified code]
```

**Targeted usage:**
```
User: "Can you simplify the scaling engine code?"
Claude: [Launches code-simplifier agent with prompt: "Simplify the scaling engine code (scalingengine/ directory)..."]
```

**After making changes:**
```
User: "I just updated the API server, can you clean it up?"
Claude: [Launches code-simplifier agent with prompt: "Simplify the recently modified API server code..."]
```

## What the Agent Does

The agent will:
- Reduce unnecessary complexity and nesting
- Improve variable and function names
- Eliminate redundant code
- Apply project coding standards
- Avoid nested ternaries (use if/else or switch)
- Choose clarity over brevity

The agent will NOT:
- Change functionality or behavior
- Alter test coverage
- Modify public APIs
- Over-simplify at the cost of clarity

## Output

After completion:
- Summary of simplified files
- Description of changes
- Edits already applied via Edit tool

## Tips

- Commit code before running /simplify for easy review
- Run `make test` after simplification to verify functionality
- Review changes with `git diff` before committing
- Agent focuses on recently modified code by default
