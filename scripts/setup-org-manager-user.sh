#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function cf_org_manager_login() {
	step "login to cf as org manager"
	# shellcheck disable=SC2154 # system_domain sourced from vars.source.sh
	cf api "https://api.${system_domain}" --skip-ssl-validation
	cf auth "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG_MANAGER_PASSWORD}"
}

function setup_space_roles_as_org_manager() {
	step "Granting space roles as org manager"

	cf_org_manager_login
	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
	cf set-space-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}" SpaceManager
	cf set-space-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}" SpaceDeveloper

	step "Setup complete!"
}

function main() {
	bbl_login
	setup_space_roles_as_org_manager
	return 0
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
