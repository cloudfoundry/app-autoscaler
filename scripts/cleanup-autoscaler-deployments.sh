#!/bin/bash
# TODO: Maybe we could give orgs from PRs like three days before they are cleaned up,
#       so your deployment from Friday is still around on Monday.

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

# Shared orgs used by multiple PRs — must never be cleaned up by this script.
# These orgs are long-lived and other deployments depend on them existing.
PROTECTED_ORGS=("SAP_autoscaler_tests_OSS")

function is_protected_org(){
	local name="$1"
	local org
	for org in "${PROTECTED_ORGS[@]}"; do
		[[ "${name}" == "${org}" ]] && return 0
	done
	return 1
}

function get_autoscaler_deployments(){
	cf curl /v3/organizations | jq -r '.resources[] | select(.name | contains("autoscaler")) | .name' \
		| while IFS= read -r org; do
			is_protected_org "${org}" || echo "${org}"
		done
}

function main(){
	bbl_login
	cf_login
	local deployments
	deployments=$(get_autoscaler_deployments)
	step "Deployments to cleanup: ${deployments}"
	while IFS='' read -r deployment
	do
		unset_vars
		export DEPLOYMENT_NAME="${deployment}"
		# shellcheck source=scripts/vars.source.sh
		source "${script_dir}/vars.source.sh"

		cleanup_acceptance_run
		cleanup_service_broker
		cleanup_credhub
	  cleanup_apps
	done <<< "${deployments}"

}

main "$@"
