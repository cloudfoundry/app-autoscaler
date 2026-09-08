#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

if [[ -z "${POSTGRES_SERVICE_OFFERING:-}" ]]; then
  echo "ERROR: POSTGRES_SERVICE_OFFERING is not set" >&2
  exit 1
fi
if [[ -z "${POSTGRES_SERVICE_PLAN:-}" ]]; then
  echo "ERROR: POSTGRES_SERVICE_PLAN is not set" >&2
  exit 1
fi

cf_deployment_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

echo "Creating service '${DEPLOYMENT_NAME}' (${POSTGRES_SERVICE_OFFERING}/${POSTGRES_SERVICE_PLAN})"
cf create-service "${POSTGRES_SERVICE_OFFERING}" "${POSTGRES_SERVICE_PLAN}" "${DEPLOYMENT_NAME}"
