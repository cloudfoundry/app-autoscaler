---
name: simplify
description: Simplify and refine code for clarity and maintainability
allowed-tools:
  - Task
---

# Code Simplifier

This skill simplifies and refines code for clarity, consistency, and maintainability while preserving all functionality.

## Usage

When invoked, this skill will:
1. Launch the code-simplifier agent
2. The agent will analyze recently modified code (or code you specify)
3. Apply refinements to improve clarity, consistency, and maintainability
4. Preserve all functionality - only the code structure improves

## How it Works

The skill uses the Task tool to launch the **code-simplifier** agent (subagent_type: "code-simplifier"), which:
- Focuses on recently modified code by default
- Applies project-specific best practices from CLAUDE.md
- Enhances clarity by reducing complexity and improving naming
- Never changes functionality - only improves code structure

## Instructions

When this skill is invoked:

1. **Launch the code-simplifier agent** using the Task tool with:
   - `subagent_type: "code-simplifier"`
   - `description: "Simplify and refine code"`
   - `prompt`: Provide context about what code to simplify:
     - If user specified files/directories, include that in the prompt
     - If no scope given, tell agent to focus on recently modified code
     - Example prompt: "Simplify the recently modified code, focusing on clarity and maintainability while preserving all functionality"

2. **Pass relevant context** in the prompt:
   - Mention any specific files or components the user wants simplified
   - Include any specific concerns (e.g., "the reviewer said it's too complex")
   - For this autoscaler project, key areas: api, eventgenerator, scalingengine, metricsforwarder, operator

3. **Let the agent work autonomously** - it has access to all tools and will:
   - Identify target files
   - Read and analyze code
   - Apply simplifications
   - Use Edit tool to refine code

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

The code-simplifier agent will:
- ✓ Reduce unnecessary complexity and nesting
- ✓ Improve variable and function names for clarity
- ✓ Eliminate redundant code
- ✓ Apply project coding standards
- ✓ Avoid nested ternaries (use if/else or switch instead)
- ✓ Choose clarity over brevity

The agent will NOT:
- ✗ Change functionality or behavior
- ✗ Alter test coverage
- ✗ Modify public APIs
- ✗ Over-simplify at the cost of clarity

## Output

After the agent completes, you'll see:
- Summary of files that were simplified
- Description of changes made
- The agent will have already applied edits using the Edit tool

## Tips

- Commit your code before running /simplify to easily review changes
- Run `make test` after simplification to verify functionality
- Review the changes with `git diff` before committing
- The agent focuses on recently modified code by default - specify scope if needed
