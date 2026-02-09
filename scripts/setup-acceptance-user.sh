#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function setup_acceptance_user() {
	step "Setting up acceptance test user"
	log "Organization: ${AUTOSCALER_ORG}"
	log "Username: ${AUTOSCALER_ORG_MANAGER_USER}"

	# Generate/retrieve password from CredHub
	credhub generate --no-overwrite -n "${CREDHUB_ORG_MANAGER_PASSWORD_PATH}" --length 32 -t password > /dev/null
	local password
	password=$(credhub get --quiet --name="${CREDHUB_ORG_MANAGER_PASSWORD_PATH}")

	# Delete and recreate user for idempotency
	cf delete-user -f "${AUTOSCALER_ORG_MANAGER_USER}" || true
	cf create-user "${AUTOSCALER_ORG_MANAGER_USER}" "${password}"

	# Assign OrgManager role
	cf set-org-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" OrgManager

	step "Setup complete!"
	log "User: ${AUTOSCALER_ORG_MANAGER_USER}"
	log "Password stored in CredHub at: ${CREDHUB_ORG_MANAGER_PASSWORD_PATH}"
}

function main() {
	bbl_login
	cf_admin_login
	setup_acceptance_user
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
