#!/usr/bin/env bash
# shellcheck disable=SC2086

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

bbl_login
cf_org_manager_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
