#! /usr/bin/env bash
#
# To run this script you need to have set up the target using
# fly login -t app-autoscaler -c https://concourse.app-runtime-interfaces.ci.cloudfoundry.org -n app-autoscaler
#
# When running concourse locally: ` fly -t "local" login -c "http://localhost:8080" `
# Then  `TARGET=local set-pipeline.sh`
set -euo pipefail

SCRIPT_RELATIVE_DIR=$(dirname "${BASH_SOURCE[0]}")
pushd "${SCRIPT_RELATIVE_DIR}" > /dev/null
  TARGET="${TARGET:-app-autoscaler}"
  FLY_OPTS="${FLY_OPTS:-}"

  PIPELINE_NAME="infrastructure"

  # shellcheck disable=SC2086
  fly -t "${TARGET}" set-pipeline --config="pipeline.yml" --pipeline="${PIPELINE_NAME}" ${FLY_OPTS}
popd
