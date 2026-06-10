#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function setup_security_group() {
	local name="$1"
	local file="$2"
	echo "Setting up security group '${name}' for org '${AUTOSCALER_ORG}' space '${AUTOSCALER_SPACE}'"
	cf create-security-group "${name}" "${file}" || true
	cf update-security-group "${name}" "${file}"
	cf bind-security-group "${name}" "${AUTOSCALER_ORG}" --space "${AUTOSCALER_SPACE}"
	echo "Security group '${name}' configured successfully for space '${AUTOSCALER_SPACE}'"
}

function main() {
	bbl_login
	cf_admin_login
	setup_security_group "metricsforwarder" "${autoscaler_dir}/metricsforwarder/security-group.json"
	return 0
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
