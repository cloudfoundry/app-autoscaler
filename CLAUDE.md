# Claude Code Best Practices for This Project

## CI/CD Workflows

### Always Monitor CI Runs
When making changes that trigger CI/CD pipelines:
1. **Always monitor the CI run** after pushing changes
2. Use `gh pr checks <pr-number>` to check status
3. Use `gh run watch <run-id>` to watch in real-time
4. Investigate failures immediately by checking logs
5. Don't wait for user to ask - proactively report CI status

### Checking CI Logs
```bash
# List recent runs
gh run list --branch <branch-name> --limit 5

# Watch a specific run
gh run watch <run-id> --exit-status

# Get logs from failed job
gh api repos/owner/repo/actions/jobs/<job-id>/logs

# Check PR status
gh pr checks <pr-number>
```

## Acceptance Testing

### OrgManager vs Admin Permissions
- **OrgManager** users can:
  - Create/delete spaces within their organization
  - Assign roles within their organization
  - **Cannot** enable service access (requires admin)
  - **Cannot** create users globally (requires admin)

- Service access must be enabled **before** tests run
- Service broker must be **registered** before enabling service access
- Use `SKIP_SERVICE_ACCESS_MANAGEMENT=true` when OrgManager can't manage service access

### Test Configuration
Key environment variables for acceptance tests:
- `USE_EXISTING_USER=true` - Skip user creation, use existing user
- `EXISTING_USER` - Username for test execution
- `USE_EXISTING_ORGANIZATION=true` - Use existing org
- `SKIP_SERVICE_ACCESS_MANAGEMENT=true` - Don't try to enable service access during tests
- `KEEP_USER_AT_SUITE_END=true` - Don't delete user after tests

## Git Workflow

### Before Pushing
1. Check what's changed: `git diff`
2. Verify commits: `git log --oneline -n 5`
3. Check remote state: `git fetch && git log origin/<branch> --oneline -n 5`
4. Pull if needed: `git pull --rebase`

### Handling Conflicts
```bash
# If rebase fails
git rebase --abort

# Reset to remote
git fetch
git reset --hard origin/<branch>

# Reapply changes
git cherry-pick <commit-sha>
```

## Debugging

### Common Issues
1. **Service not found** - Service broker not registered yet
2. **Permission denied** - User doesn't have required role
3. **User creation fails** - Only admins can create users
4. **Service access fails** - Only admins can enable service access

### Investigation Steps
1. Check timing - Is service registered before access is enabled?
2. Check user role - Does user have required permissions?
3. Check logs - What's the exact error message?
4. Check CF CLI help - `cf <command> --help`
