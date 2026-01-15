#!/usr/bin/env bash

set -euo pipefail

echo "Running $0"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

# Login to BOSH if BBL_STATE_PATH is set
if [ -n "${BBL_STATE_PATH:-}" ]; then
  bbl_login
fi

# Login to CF
cf_login

# Target CF org and space
cf_target "${autoscaler_org:-}" "${autoscaler_space:-}"

echo "Done"