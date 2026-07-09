#!/bin/bash
# TODO: Maybe we could give orgs from PRs like three days before they are cleaned up,
#       so your deployment from Friday is still around on Monday.

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function get_autoscaler_deployments(){
	cf curl /v3/organizations | jq -r '.resources[] | select(.name | contains("autoscaler")) | .name'
}

function cleanup_deployment(){
	unset_vars
	export DEPLOYMENT_NAME="$1"
	# shellcheck source=scripts/vars.source.sh
	source "${script_dir}/vars.source.sh"
	step "[${DEPLOYMENT_NAME}] starting cleanup"
	cleanup_acceptance_run
	cleanup_service_broker
	cleanup_credhub
	cleanup_apps
	step "[${DEPLOYMENT_NAME}] done"
}

function main(){
	bbl_login
	cf_login
	local deployments pids=() pid exit_code=0
	deployments=$(get_autoscaler_deployments)
	step "Deployments to cleanup: ${deployments}"

	while IFS='' read -r deployment; do
		cleanup_deployment "${deployment}" &
		pids+=($!)
	done <<< "${deployments}"

	for pid in "${pids[@]}"; do
		wait "${pid}" || exit_code=$?
	done
	return "${exit_code}"
}

main "$@"
