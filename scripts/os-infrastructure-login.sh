#! /usr/bin/env bash
# shellcheck disable=SC2086
set -eu -o pipefail

script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
# shellcheck disable=SC1091
source "${script_dir}/vars.source.sh"
# shellcheck disable=SC1091
source "${script_dir}/common.sh"


bbl_login "${BBL_STATE_PATH}"
cf_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
