#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function create_org_manager_user() {
	step "Creating org manager CF user"
	log "Organization: ${AUTOSCALER_ORG}"

	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

	local password
	password="$(openssl rand -base64 32)"

	cf delete-user -f "${AUTOSCALER_ORG_MANAGER_USER}" || true
	cf create-user "${AUTOSCALER_ORG_MANAGER_USER}" "${password}"
	cf set-org-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" OrgManager

	local existing_org="${EXISTING_ORGANIZATION:-}"
	if [[ -n "${existing_org}" && "${existing_org}" != "${AUTOSCALER_ORG}" ]]; then
		log "Granting OrgManager role in existing org: ${existing_org}"
		cf set-org-role "${AUTOSCALER_ORG_MANAGER_USER}" "${existing_org}" OrgManager
	fi

	local repo
	repo="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"

	log "Writing username to GitHub repo variable AUTOSCALER_ORG_MANAGER_USER"
	gh variable set AUTOSCALER_ORG_MANAGER_USER --body "${AUTOSCALER_ORG_MANAGER_USER}" --repo "${repo}"

	log "Writing password to GitHub repo secret AUTOSCALER_ORG_MANAGER_PASSWORD"
	gh secret set AUTOSCALER_ORG_MANAGER_PASSWORD --body "${password}" --repo "${repo}"

	step "Org manager user created and credentials stored!"
}

function main() {
	bbl_login
	cf_admin_login
	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
	local repo
	repo="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
	create_cf_test_user "${repo}" "${AUTOSCALER_ORG_MANAGER_USER}" AUTOSCALER_ORG_MANAGER_USER AUTOSCALER_ORG_MANAGER_PASSWORD "Creating org manager CF user"
	create_cf_test_user "${repo}" "${AUTOSCALER_OTHER_USER}" AUTOSCALER_OTHER_USER AUTOSCALER_OTHER_USER_PASSWORD "Creating other-user for acceptance tests"
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
