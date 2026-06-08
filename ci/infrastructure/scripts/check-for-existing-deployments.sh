#!/bin/bash
# shellcheck disable=SC2086
set -euo pipefail

DEBUG="${DEBUG:-}"
bbl_state_path="${BBL_STATE_PATH:-bbl-state/bbl-state}"

[ -n "${DEBUG}" ] && set -x



pushd "${bbl_state_path}" > /dev/null
  eval "$(bbl print-env)"
popd > /dev/null


deployment_count="$(bosh deployments --json | jq -r ".Tables[0].Rows | .[] | .name" | wc -l | sed 's/^ *//g')"

if [ "$deployment_count" != "0" ]; then
  echo "Cannot destroy infrastructure: delete $deployment_count existing deployments"
  exit 1
fi
