# App-Autoscaler release process

1. **Create a draft GitHub release (`app-autoscaler`)**
    - Run the “[create draft release][1]” job/script.
    - This job prepares the draft release but does not yet generate final artifacts; those are created only once the draft is promoted.

2. **Promote the draft release (`app-autoscaler`)**
    - Run the “[promote draft][2]” job/script.
    - If no draft exists, the job fails with “no draft release found to promote”; otherwise it promotes the draft, builds the MTAR and related artifacts, computes checksums and updates the final GitHub release assets.
    - As part of this promotion step, the version reference in the API (single version file) is patched and a commit is pushed to the target branch (usually `main`, sometimes a feature/fix branch).

3. **PR to the `app-autoscaler-release` repository**
    - The final release job uses a GitHub token (`parent repo dispatch token`) to emit a repository-dispatch event to the `app-autoscaler-release` repository.
    - On the `app-autoscaler-release` side, this event triggers a workflow that creates a PR bumping the submodule to the latest tagged commit of App-Autoscaler.
    - Optionally, you can add labels (for example for auto-merge) to let this PR merge automatically once checks pass.

4. **Run Concourse pipeline on `app-autoscaler-release`**
    - After the submodule-bump PR is merged in `app-autoscaler-release`, the Concourse pipeline picks up the change as a normal release PR.
    - Concourse runs performance tests, upgrade tests, and the release job.
    - The release job creates the GitHub release in `app-autoscaler-release`, uploads the App-Autoscaler release artifacts (including MTAR and acceptance tests) and pushes them to the configured Google Cloud Storage buckets.

[1]: https://github.com/cloudfoundry/app-autoscaler/actions/workflows/release-draft.yaml
[2]: https://github.com/cloudfoundry/app-autoscaler/actions/workflows/release-promote.yaml
