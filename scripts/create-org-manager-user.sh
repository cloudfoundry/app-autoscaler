#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

AUTOSCALER_ORG_MANAGER_USER="${AUTOSCALER_ORG_MANAGER_USER:-org-manager-user}"
AUTOSCALER_OTHER_USER="${AUTOSCALER_OTHER_USER:-other-user}"

function create_org_manager_user() {
	local repo="$1"
	step "Creating org manager CF user"
	log "Organization: ${AUTOSCALER_ORG}"

	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

	local password
	password="$(openssl rand -base64 32)"

	cf delete-user -f "${AUTOSCALER_ORG_MANAGER_USER}" || true
	cf create-user "${AUTOSCALER_ORG_MANAGER_USER}" "${password}"
	cf set-org-role "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG}" OrgManager

	log "Writing username to GitHub repo variable AUTOSCALER_ORG_MANAGER_USER"
	gh variable set AUTOSCALER_ORG_MANAGER_USER --body "${AUTOSCALER_ORG_MANAGER_USER}" --repo "${repo}"

	log "Writing password to GitHub repo secret AUTOSCALER_ORG_MANAGER_PASSWORD"
	gh secret set AUTOSCALER_ORG_MANAGER_PASSWORD --body "${password}" --repo "${repo}"

	step "Org manager user created and credentials stored!"
}

function create_other_user() {
	local repo="$1"
	step "Creating other-user for acceptance tests"
	log "Organization: ${AUTOSCALER_ORG}"

	local password
	password="$(openssl rand -base64 32)"

	cf delete-user -f "${AUTOSCALER_OTHER_USER}" || true
	cf create-user "${AUTOSCALER_OTHER_USER}" "${password}"
	cf set-org-role "${AUTOSCALER_OTHER_USER}" "${AUTOSCALER_ORG}" OrgManager

	log "Writing username to GitHub repo variable AUTOSCALER_OTHER_USER"
	gh variable set AUTOSCALER_OTHER_USER --body "${AUTOSCALER_OTHER_USER}" --repo "${repo}"

	log "Writing password to GitHub repo secret AUTOSCALER_OTHER_USER_PASSWORD"
	gh secret set AUTOSCALER_OTHER_USER_PASSWORD --body "${password}" --repo "${repo}"

	step "Other user created and credentials stored!"
}

function main() {
	bbl_login
	cf_login
	local repo
	repo="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
	create_org_manager_user "${repo}"
	create_other_user "${repo}"
	return 0
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
