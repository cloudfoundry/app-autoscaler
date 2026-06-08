#!/bin/bash
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
set -euo pipefail

function bosh_login(){
  if [[ ! -d ${bbl_state_path} ]]; then
    echo "FAILED: Did not find bbl-state folder at ${bbl_state_path}"
    echo "Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH";
    exit 1;
  fi

  pushd "${bbl_state_path}" > /dev/null
    eval "$(bbl print-env)"
  popd > /dev/null
}

function cf_login(){
  cf api "https://api.${system_domain}" --skip-ssl-validation
  CF_ADMIN_PASSWORD=$(credhub get -n /bosh-autoscaler/cf/cf_admin_password -q)
  cf auth admin "$CF_ADMIN_PASSWORD"
}
