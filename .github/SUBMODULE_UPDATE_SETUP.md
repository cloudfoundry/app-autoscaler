# Automatic Submodule Update Setup

This repository (app-autoscaler) automatically notifies the parent repository (app-autoscaler-release) when a new release is published.

## How it works

1. When a release is published in `app-autoscaler`, the workflow `.github/workflows/dispatch-release-to-parent.yaml` triggers
2. It sends a `repository_dispatch` event to `app-autoscaler-release` with release information
3. The parent repo receives the event and creates a PR to update the submodule

## Setup Required in app-autoscaler-release

Create the file `.github/workflows/update-autoscaler-submodule.yaml`:

```yaml
---
name: Update Autoscaler Submodule

on:
  repository_dispatch:
    types: [autoscaler-release-published]

permissions:
  contents: write
  pull-requests: write

jobs:
  update-submodule:
    runs-on: ubuntu-latest
    name: Update autoscaler submodule to new release
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          submodules: true
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Update submodule to release tag
        run: |
          cd src/autoscaler
          git fetch origin
          git checkout ${{ github.event.client_payload.release_tag }}
          cd ../..
          git add src/autoscaler

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v6
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: |
            Update autoscaler submodule to ${{ github.event.client_payload.release_tag }}

            Release: ${{ github.event.client_payload.release_url }}
          branch: update-autoscaler-${{ github.event.client_payload.release_tag }}
          delete-branch: true
          title: "Update autoscaler to ${{ github.event.client_payload.release_tag }}"
          body: |
            ## Autoscaler Submodule Update

            This PR updates the `src/autoscaler` submodule to the newly released version.

            **Release:** [${{ github.event.client_payload.release_tag }}](${{ github.event.client_payload.release_url }})
            **Commit:** ${{ github.event.client_payload.commit_sha }}

            ### Changes
            - Updates `src/autoscaler` submodule reference

            🤖 This PR was automatically created by the release workflow.
          labels: |
            dependencies
            autoscaler-update
            automated
```

## Required Secrets

### In app-autoscaler (this repo):
- **`PARENT_REPO_DISPATCH_TOKEN`**: A GitHub Personal Access Token (PAT) with `repo` scope
  - The token must have write access to `cloudfoundry/app-autoscaler-release`
  - Used to send the repository_dispatch event

### To create the PAT:
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token with `repo` scope (or use a fine-grained token with `contents: write` on app-autoscaler-release)
3. Add it as a repository secret named `PARENT_REPO_DISPATCH_TOKEN` in this repo's settings

## Testing

You can test the workflow by:
1. Creating a test release in this repository
2. Checking the Actions tab to see if the dispatch was sent
3. Checking the parent repo's Actions tab to see if it received the event and created a PR
