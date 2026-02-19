#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function create_cf_user() {
	local username="$1"
	local credhub_path="$2"

	log "Creating user: ${username}"
	credhub generate --no-overwrite -n "${credhub_path}" --length 32 -t password > /dev/null
	local password
	password=$(credhub get --quiet --name="${credhub_path}")

	cf delete-user -f "${username}" || true
	cf create-user "${username}" "${password}"
}

function setup_acceptance_users() {
	step "Setting up acceptance test users"
	log "Organization: ${AUTOSCALER_ORG}"

	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

	create_cf_user "${AUTOSCALER_ORG_MANAGER_USER}" "${CREDHUB_ORG_MANAGER_PASSWORD_PATH}"
	cf set-org-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" OrgManager
	cf set-space-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}" SpaceManager
	cf set-space-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}" SpaceDeveloper

	create_cf_user "${AUTOSCALER_OTHER_USER}" "${CREDHUB_OTHER_USER_PASSWORD_PATH}"

	step "Setup complete!"
}

function main() {
	bbl_login
	cf_admin_login
	setup_acceptance_users
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
