#!/bin/bash

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function get_autoscaler_deployments(){
	cf curl /v3/organizations | jq -r '.resources[] | select(.name | contains("autoscaler")) | .name'
}

function main(){
	cf_login
	local deployments=$(get_autoscaler_deployments)
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
