#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function main() {
	step "cleaning up deployment ${DEPLOYMENT_NAME}"

	if is_oss_infrastructure; then
		bbl_login
	fi
	cf_deployment_login

	cleanup_apps
	cleanup_acceptance_run
	cleanup_service_broker

	if is_oss_infrastructure; then
		cleanup_credhub
		cleanup_db
	else
		step "deleting CF service '${DEPLOYMENT_NAME}'"
		cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
		cf delete-service -f "${DEPLOYMENT_NAME}" || echo " - could not delete service '${DEPLOYMENT_NAME}'"
	fi
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
