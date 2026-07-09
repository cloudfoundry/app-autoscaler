# App-autoscaler team and pipelines management for Concourse

Concourse URL: <https://concourse.app-runtime-interfaces.ci.cloudfoundry.org/>

## Dependencies

None. Terraform scripts are contained with terragrunt config.

## Requirements

Add fly target `app-autoscaler-release  https://concourse.app-runtime-interfaces.ci.cloudfoundry.org`

Login to your fly target prior to executing terragrunt

## Usage

```sh
terragrunt plan
terragrunt apply
```
