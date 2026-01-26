#!/usr/bin/env bash

set -euo pipefail

echo "Running $0"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

if [ -n "${BBL_STATE_PATH:-}" ]; then
	bbl_login
fi

cf_login
cf_target "${autoscaler_org}" "${autoscaler_space}"

echo "Done"