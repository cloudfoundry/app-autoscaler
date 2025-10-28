#! /usr/bin/env bash

# 🪧 NOTE: to turn on debug use DEBUG=true
# shellcheck disable=SC2155,SC2034
#

if [ -z "${BASH_SOURCE[0]}" ]; then
	echo  "### Source this from inside a script only! "
	echo  "### ======================================="
	echo
	return
fi

write_error_state() {
	echo "Error failed execution of \"$1\" at line $2"
	local frame=0
	while true ; do
		caller $frame && break
		((frame++));
	done
}

trap 'write_error_state "$BASH_COMMAND" "$LINENO"' ERR

debug=${DEBUG:-}
if [ -n "${debug}" ] && [ ! "${debug}" = "false" ]; then
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

script_dir="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir=$(realpath -e "${script_dir}/../../..")

# This environment-variable is used as the target-name for concourse that is used to communicate
# with the concourse-instance that manages the os-pipelines of this repository under
# <concourse.app-runtime-interfaces.ci.cloudfoundry.org>
CONCOURSE_AAS_RELEASE_TARGET="${CONCOURSE_AAS_RELEASE_TARGET:-app-autoscaler-release}"
debug "CONCOURSE_AAS_RELEASE_TARGET: ${CONCOURSE_AAS_RELEASE_TARGET}"

export PR_NUMBER=${PR_NUMBER:-$(gh pr view --json number --jq '.number' )}
debug "PR_NUMBER: '${PR_NUMBER}'"
user=${USER:-"test"}


export DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-"autoscaler-mta-${PR_NUMBER}"}"
[ "${DEPLOYMENT_NAME}" = "autoscaler-mta" ] && DEPLOYMENT_NAME="${user}"

debug "DEPLOYMENT_NAME: ${DEPLOYMENT_NAME}"
log "set up vars: DEPLOYMENT_NAME=${DEPLOYMENT_NAME}"
deployment_name="${DEPLOYMENT_NAME}"

export AUTOSCALER_ORG="${AUTOSCALER_ORG:-$DEPLOYMENT_NAME}"
debug "AUTOSCALER_ORG: ${AUTOSCALER_ORG}"
log "set up vars: AUTOSCALER_ORG=${AUTOSCALER_ORG}"
autoscaler_org="${AUTOSCALER_ORG}"

export AUTOSCALER_SPACE="${AUTOSCALER_SPACE:-$DEPLOYMENT_NAME}"
debug "AUTOSCALER_SPACE: ${AUTOSCALER_SPACE}"
log "set up vars: AUTOSCALER_SPACE=${AUTOSCALER_SPACE}"
autoscaler_space="${AUTOSCALER_SPACE}"

export SYSTEM_DOMAIN="${SYSTEM_DOMAIN:-"autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"}"
debug "SYSTEM_DOMAIN: ${SYSTEM_DOMAIN}"
system_domain="${SYSTEM_DOMAIN}"

BBL_STATE_PATH="${BBL_STATE_PATH:-$( realpath "${root_dir}/../app-autoscaler-env-bbl-state/bbl-state" )}"
# We want to print out the name of the variable literally and marked as shell-variable, therefore:
# shellcheck disable=SC2016
BBL_STATE_PATH="$(realpath --canonicalize-existing "${BBL_STATE_PATH}" \
									|| echo 'ERR_invalid_state_path, please set ${BBL_STATE_PATH}' )"
export BBL_STATE_PATH
debug "BBL_STATE_PATH: ${BBL_STATE_PATH}"

AUTOSCALER_DIR="${AUTOSCALER_DIR:-${root_dir}}"
export AUTOSCALER_DIR="$(realpath -e "${AUTOSCALER_DIR}" )"
debug "AUTOSCALER_DIR: ${AUTOSCALER_DIR}"
autoscaler_dir="${AUTOSCALER_DIR}"

AUTOSCALER_ACCEPTANCE_DIR="${AUTOSCALER_ACCEPTANCE_DIR:-${script_dir}/../acceptance}"
export AUTOSCALER_ACCEPTANCE_DIR="$(realpath -e "${AUTOSCALER_ACCEPTANCE_DIR}" )"
debug "AUTOSCALER_ACCEPTANCE_DIR: ${AUTOSCALER_ACCEPTANCE_DIR}"
autoscaler_acceptance_dir="${AUTOSCALER_ACCEPTANCE_DIR}"

export SERVICE_NAME="${DEPLOYMENT_NAME}"
debug "SERVICE_NAME: ${SERVICE_NAME}"
service_name="%{SERVICE_NAME"

export SERVICE_BROKER_NAME="${DEPLOYMENT_NAME}servicebroker"
debug "SERVICE_BROKER_NAME: ${SERVICE_BROKER_NAME}"
service_broker_name="${SERVICE_BROKER_NAME}"

export NAME_PREFIX="${NAME_PREFIX:-"${DEPLOYMENT_NAME}-TESTS"}"
debug "NAME_PREFIX: ${NAME_PREFIX}"
name_prefix="${NAME_PREFIX}"

export GINKGO_OPTS=${GINKGO_OPTS:-"--fail-fast"}

export PERFORMANCE_APP_COUNT="${PERFORMANCE_APP_COUNT:-50}"
debug "PERFORMANCE_APP_COUNT: ${PERFORMANCE_APP_COUNT}"

export PERFORMANCE_APP_PERCENTAGE_TO_SCALE="${PERFORMANCE_APP_PERCENTAGE_TO_SCALE:-30}"
debug "PERFORMANCE_APP_PERCENTAGE_TO_SCALE: ${PERFORMANCE_APP_PERCENTAGE_TO_SCALE}"

export PERFORMANCE_SETUP_WORKERS="${PERFORMANCE_SETUP_WORKERS:-20}"
debug "PERFORMANCE_SETUP_WORKERS: ${PERFORMANCE_SETUP_WORKERS}"

export PERFORMANCE_TEARDOWN=${PERFORMANCE_TEARDOWN:-true}
debug "PERFORMANCE_TEARDOWN: ${PERFORMANCE_TEARDOWN}"

export PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA=${PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA:-false}
debug "PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA: ${PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA}"

export CPU_UPPER_THRESHOLD=${CPU_UPPER_THRESHOLD:-100}
debug "CPU_UPPER_THRESHOLD: ${CPU_UPPER_THRESHOLD}"
cpu_upper_threshold=${CPU_UPPER_THRESHOLD}

