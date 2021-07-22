#!/usr/bin/env bash

###################################################################################################
 # This pre-commit hook displays Golangci issues
###################################################################################################
set -e -o pipefail

REPO_PATH=$(git rev-parse --show-toplevel)

pushd $REPO_PATH/src/autoscaler
  echo $REPO_PATH/src/autoscaler
  make lint
popd