---
name: unresolved-comments
description: Fetch and display only unresolved comments from a GitHub pull request
allowed-tools:
  - Bash(gh pr view*)
  - Bash(gh api*)
---

You are an AI assistant integrated into a git-based version control system. Your task is to fetch and display ONLY unresolved comments from a GitHub pull request.

Follow these steps:

1. Use `gh pr view --json number,headRepository` to get the PR number and repository info
2. Use GitHub's GraphQL API to fetch review threads with their resolution status:
   ```
   gh api graphql -f query='
   {
     repository(owner: "{owner}", name: "{repo}") {
       pullRequest(number: {number}) {
         reviewThreads(first: 100) {
           nodes {
             isResolved
             isOutdated
             comments(first: 50) {
               nodes {
                 author { login }
                 body
                 path
                 line
                 originalLine
                 diffHunk
                 createdAt
               }
             }
           }
         }
       }
     }
   }'
   ```
3. Filter to show ONLY threads where `isResolved: false`
4. Parse and format all unresolved comments in a readable way
5. Return ONLY the formatted unresolved comments, with no additional text

Format the comments as:

## Unresolved Comments

[For each unresolved comment thread:]
- @author file.ts#line:
  ```diff
  [diff_hunk from the API response]
  ```
  > quoted comment text

  [any replies indented]

If there are no unresolved comments, return "No unresolved comments found."

Important notes:
- Use GitHub's GraphQL API which provides the `isResolved` field for review threads
- Only show threads where `isResolved: false`
- PR-level comments (not attached to code) are not part of review threads, so they won't appear here
- Preserve the threading/nesting of comment replies within unresolved threads
- Show the file and line number context for code review comments
- The `isOutdated` field indicates if the code has changed since the comment was made - you may want to note this

Remember:
1. Only show unresolved comments, no explanatory text
2. Focus on code review comments (review threads)
3. Preserve the threading/nesting of comment replies (all comments within the same thread node)
4. Show the file and line number context
5. Use jq to parse the JSON responses from the GraphQL API
