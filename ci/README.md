# app-autoscaler-ci

This repository provides all public scripts and pipeline deployments used
by the app autoscaler team.  The public pipeline is hosted at: <https://concourse.app-runtime-interfaces.ci.cloudfoundry.org>.

To reproduce this pipeline, you can use your own private configuration files for the `pipeline.yml` files as described below.

## Autoscaler

This directory contains the concourse `pipeline.yml` for the autoscaler [pipeline](https://concourse.app-runtime-interfaces.ci.cloudfoundry.org/teams/app-autoscaler/pipelines/app-autoscaler-release)
and all of the associated scripts. To use this manifest, you need to provide a private configuration file
for all of the template parameters.

NOTE: If you are recreating this pipeline, for personal use and do not have authority to update
tracker or push to github. The `pipeline.yml` file needs to have any `tracker` sections commented
out as well as the app-autoscaler private key

## dockerfiles

These docker images in this repo are built and pushed with GitHub actions, they are hosted on ghcr.io

## Terrgrunt

This directory contains the terragrunt managed stacks of resouces in account app-runtime-interfaces-wg GCP project.

## Deploy pipeline

__Setup__

```
make set-target
make set-autoscaler-pipeline
```

## Unpause pipeline and jobs

```
# You will be prompted to select the specific jobs you want to unpause.
make unpause-pipeline
```
