# App-Autoscaler release process

1. **Create a draft GitHub release**
    - Run the [create draft release][1] job/script.
    - This job prepares the draft release but does not yet generate final artifacts; those are created only once the draft is promoted.

2. **Review the draft release**
    - Review the draft release on GitHub, check the version, description, and other metadata.
    - If any changes are needed, update the draft release on GitHub and repeat this step until the draft is ready for promotion.

3. **Promote the draft release**
    - Run the [promote draft][2] job/script.
    - If no draft exists, the job fails with “no draft release found to promote”; otherwise it promotes the draft, builds the MTAR and related artifacts, computes checksums and updates the final GitHub release assets.
    - As part of this promotion step, the version reference in [`api/default_info.json`](/api/default_info.json) is patched and a commit is pushed to the target branch (usually `main`, sometimes a feature/fix branch).

[1]: https://github.com/cloudfoundry/app-autoscaler/actions/workflows/release-draft.yaml
[2]: https://github.com/cloudfoundry/app-autoscaler/actions/workflows/release-promote.yaml
