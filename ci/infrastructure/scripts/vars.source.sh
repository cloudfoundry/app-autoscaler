#!/bin/bash
# Source this file please.
# NOTE: to turn on debug use DEBUG=true
# shellcheck disable=SC2155
if [ -z "${BASH_SOURCE[0]}" ]; then
  echo  "### Source this from inside a script only! "
  echo  "### ======================================="
  echo
  return
fi

debug=${DEBUG:-}
if [ -n "${debug}" ]; then
  function debug(){ echo "  -> $1"; }
else
  function debug(){ :; }
fi

function warn(){
  echo " - WARN: $1"
}

function log(){
  echo " - $1"
}

function step(){
  echo "# $1"
}

script_dir="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir=$(realpath -e "${script_dir}/../../..")

BBL_STATE_PATH="${BBL_STATE_PATH:-$( realpath -e "${root_dir}/../app-autoscaler-env-bbl-state/bbl-state" 2> /dev/null || realpath -e "${root_dir}/../bbl-state/bbl-state" 2> /dev/null )}"
export BBL_STATE_PATH="$(realpath -e "${BBL_STATE_PATH}" )"
export bbl_state_path="${BBL_STATE_PATH}"
debug  "BBL_STATE_PATH: ${BBL_STATE_PATH}"


AUTOSCALER_DIR="${AUTOSCALER_DIR:-${root_dir}}"
export AUTOSCALER_DIR="$(realpath -e "${AUTOSCALER_DIR}" )"
export autoscaler_dir="${AUTOSCALER_DIR}"
debug "AUTOSCALER_DIR: ${AUTOSCALER_DIR}"

CI_DIR="${CI_DIR:-$(realpath -e "${root_dir}/ci")}"
export CI_DIR="$(realpath -e "${CI_DIR}")"
debug "CI_DIR: ${CI_DIR}"
export ci_dir="${CI_DIR}"

export SYSTEM_DOMAIN="${SYSTEM_DOMAIN:-"autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"}"
debug "SYSTEM_DOMAIN: ${SYSTEM_DOMAIN}"
export system_domain="${SYSTEM_DOMAIN}"

export BOSH_USERNAME="${BOSH_USERNAME:-admin}"
debug "BOSH_USERNAME: ${BOSH_USERNAME}"
export bosh_username="${BOSH_USERNAME}"

export BBL_GCP_PROJECT_ID="${BBL_GCP_PROJECT_ID:-"cloud-foundry-310819"}"
export bbl_gcp_project_id="${BBL_GCP_PROJECT_ID}"
debug "BBL_GCP_PROJECT_ID: ${BBL_GCP_PROJECT_ID}"

export BBL_GCP_PROJECT_WG="${BBL_GCP_PROJECT_WG:-"app-runtime-interfaces-wg"}"
export bbl_gcp_project_wg="${BBL_GCP_PROJECT_WG}"
debug "BBL_GCP_PROJECT_WG: ${BBL_GCP_PROJECT_WG}"

export BBL_GCP_SERVICE_ACCOUNT_JSON="${BBL_GCP_SERVICE_ACCOUNT_JSON:-"${HOME}/.ssh/gcp.key.json"}"
export bbl_gcp_service_account_json="${BBL_GCP_SERVICE_ACCOUNT_JSON}"
debug "BBL_GCP_SERVICE_ACCOUNT_JSON: SKIPPED"

export GCP_DNS_ZONE=${GCP_DNS_ZONE:-"app-runtime-interfaces"}
export gcp_dns_zone="${GCP_DNS_ZONE}"
debug "GCP_DNS_ZONE: ${GCP_DNS_ZONE}"

export BBL_IAAS="${BBL_IAAS:-"gcp"}"
export bbl_iaas="${BBL_IAAS}"
debug "BBL_IAAS: ${BBL_IAAS}"

export BBL_ENV_NAME="autoscaler"
export bbl_env_name="${BBL_ENV_NAME}"
debug "BBL_ENV_NAME: ${BBL_ENV_NAME}"

export BBL_GCP_REGION="europe-west3"
export bbl_gcp_region="${BBL_GCP_REGION}"
debug "BBL_GCP_REGION: ${BBL_GCP_REGION}"

export BBL_GCP_ZONE="europe-west3-a"
export bbl_gcp_zone="${BBL_GCP_ZONE}"
debug "BBL_GCP_ZONE: ${BBL_GCP_ZONE}"

export CF_ORG="system"
export cf_org="${CF_ORG}"
debug "CF_ORG: ${CF_ORG}"

export CF_SPACE="production"
export cf_space="${CF_SPACE}"
debug "CF_SPACE: ${CF_SPACE}"

function unset_vars() {
  unset BOSH_USERNAME
  unset CI_DIR
  unset AUTOSCALER_DIR
  unset BBL_STATE_PATH
  unset SYSTEM_DOMAIN
  unset BBL_GCP_PROJECT_ID
  unset BBL_GCP_SERVICE_ACCOUNT_JSON
  unset GCP_DNS_ZONE
  unset BBL_IAAS
  unset BBL_ENV_NAME
  unset BBL_GCP_REGION
  unset BBL_GCP_ZONE
}
