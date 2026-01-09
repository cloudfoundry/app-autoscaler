#!/bin/bash

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# shellcheck disable=SC1091
source "${script_dir}/common.sh"

function main(){
  bbl_login "${BBL_STATE_PATH}"
  cf_login
  cleanup_acceptance_run
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
