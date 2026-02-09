
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"
# Security group configuration
#
SECURITY_GROUP_NAME="metricsforwarder"
SECURITY_GROUP_FILE="${autoscaler_dir}/metricsforwarder/security-group.json"

function validate_inputs() {
	if [ -z "${BBL_STATE_PATH:-}" ]; then
		echo "ERROR: BBL_STATE_PATH environment variable is not set. Please set it to the path of your BBL state directory."
		exit 1
	fi

}

function main() {
	bbl_login
	cf_admin_login
  echo "Setting up security group '${SECURITY_GROUP_NAME}' for org '${AUTOSCALER_ORG}' space '${AUTOSCALER_SPACE}'"

  # Create or update security group (space-scoped, not global)
  cf create-security-group "${SECURITY_GROUP_NAME}" "${SECURITY_GROUP_FILE}" || true
  cf update-security-group "${SECURITY_GROUP_NAME}" "${SECURITY_GROUP_FILE}"
  cf bind-security-group "${SECURITY_GROUP_NAME}" "${AUTOSCALER_ORG}" --space "${AUTOSCALER_SPACE}"

  echo "Security group '${SECURITY_GROUP_NAME}' configured successfully for space '${AUTOSCALER_SPACE}'"
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"


