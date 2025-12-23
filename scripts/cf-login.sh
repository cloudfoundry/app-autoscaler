#!/usr/bin/env bash

set -euo pipefail

echo "Running $0"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
source "${SCRIPT_DIR}/vars.source.sh"
source "${SCRIPT_DIR}/common.sh"

# Login to BOSH if BBL_STATE_PATH is set
if [ -n "${BBL_STATE_PATH:-}" ]; then
  if [[ ! -d "${BBL_STATE_PATH}" ]]; then
    echo "⛔ FAILED: Did not find bbl-state folder at ${BBL_STATE_PATH}" >&2
    echo 'Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH' >&2
    exit 1
  fi
  echo "# bosh login"
  eval "$("${SCRIPT_DIR}/bbl-print-env.sh" "${BBL_STATE_PATH}")"
fi

# Login to CF
cf_login

# Target CF org and space
cf_target "${autoscaler_org}" "${autoscaler_space}"

echo "Done"
