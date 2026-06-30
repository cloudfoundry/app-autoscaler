#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

if is_oss_infrastructure; then
  bbl_login
fi
cf_deployment_login

cf create-space -o "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}" || cf space "${AUTOSCALER_SPACE}" --guid >/dev/null
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
