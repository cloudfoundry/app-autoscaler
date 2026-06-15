#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

AUTOSCALER_ORG_MANAGER_USER="${AUTOSCALER_ORG_MANAGER_USER:-org-manager-user}"
AUTOSCALER_OTHER_USER="${AUTOSCALER_OTHER_USER:-other-user}"

function create_cf_test_user() {
	local repo="$1" username="$2" var_name="$3" secret_name="$4" step_msg="$5"
	step "${step_msg}"
	log "Organization: ${AUTOSCALER_ORG}"

	local password
	password="$(openssl rand -base64 32)"

	cf delete-user -f "${username}" || true
	cf create-user "${username}" "${password}"
	cf set-org-role "${username}" "${AUTOSCALER_ORG}" OrgManager

	log "Writing username to GitHub repo variable ${var_name}"
	gh variable set "${var_name}" --body "${username}" --repo "${repo}"

	log "Writing password to GitHub repo secret ${secret_name}"
	gh secret set "${secret_name}" --body "${password}" --repo "${repo}"

	step "User ${username} created and credentials stored!"
}

function main() {
	bbl_login
	cf_login
	cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"
	local repo
	repo="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
	create_cf_test_user "${repo}" "${AUTOSCALER_ORG_MANAGER_USER}" AUTOSCALER_ORG_MANAGER_USER AUTOSCALER_ORG_MANAGER_PASSWORD "Creating org manager CF user"
	create_cf_test_user "${repo}" "${AUTOSCALER_OTHER_USER}" AUTOSCALER_OTHER_USER AUTOSCALER_OTHER_USER_PASSWORD "Creating other-user for acceptance tests"
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
