#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

cf_admin_password="${CF_ADMIN_PASSWORD:-}"
skip_ssl_validation="${SKIP_SSL_VALIDATION:-true}"
use_existing_organization="${USE_EXISTING_ORGANIZATION:-false}"
existing_organization="${EXISTING_ORGANIZATION:-}"
use_existing_space="${USE_EXISTING_SPACE:-false}"
existing_space="${EXISTING_SPACE:-}"
performance_app_count="${PERFORMANCE_APP_COUNT:-}"
performance_app_percentage_to_scale="${PERFORMANCE_APP_PERCENTAGE_TO_SCALE:-}"
performance_setup_workers="${PERFORMANCE_SETUP_WORKERS:-}"
performance_update_existing_org_quota=${PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA:-true}
cpu_upper_threshold=${CPU_UPPER_THRESHOLD:-100}

if [[ -z "${cf_admin_password}" ]]
then
	bbl_login
	cf_admin_password="$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')"
fi

# Use admin user for main branch, org-manager user for PRs
if is_main_deployment; then
	autoscaler_org_manager_user="admin"
	autoscaler_org_manager_password="${cf_admin_password}"
	skip_service_access_management="false"
else
	# For PRs, use dedicated org manager user (password from GH secret)
	if [[ -z "${AUTOSCALER_ORG_MANAGER_PASSWORD:-}" ]]; then
		echo "ERROR: AUTOSCALER_ORG_MANAGER_PASSWORD is not set (required for PR deployments)" >&2
		exit 1
	fi
	autoscaler_org_manager_user="${AUTOSCALER_ORG_MANAGER_USER}"
	autoscaler_org_manager_password="${AUTOSCALER_ORG_MANAGER_PASSWORD}"
	skip_service_access_management="${SKIP_SERVICE_ACCESS_MANAGEMENT:-true}"
fi

function write_app_config() {
	local -r config_path="$1"
	local -r use_existing_organization="$2"
	local -r use_existing_space="$3"
	local -r existing_org="$4"
	local -r existing_space="$5"
	local -r existing_user="$6"
	local -r existing_user_password="$7"
	local -r other_existing_user="$8"
	local -r other_existing_user_password="$9"

	cat > "${config_path}" << EOF
{
	"api": "api.${system_domain}",
	"admin_user": "${autoscaler_org_manager_user}",
	"admin_password": "${autoscaler_org_manager_password}",
	"apps_domain": "${system_domain}",
	"skip_ssl_validation": ${skip_ssl_validation},
	"use_http": false,
	"service_name": "${deployment_name}",
	"service_plan": "autoscaler-free-plan",
	"service_broker": "${deployment_name}",
	"use_existing_organization": ${use_existing_organization},
	"existing_organization": "${existing_org}",
	"use_existing_space": ${use_existing_space},
	"existing_space": "${existing_space}",
	"existing_user": "${existing_user}",
	"existing_user_password": "${existing_user_password}",
	"other_existing_user": "${other_existing_user}",
	"other_existing_user_password": "${other_existing_user_password}",
	"skip_service_access_management": ${skip_service_access_management},
	"aggregate_interval": 120,
	"default_timeout": 60,
	"cpu_upper_threshold": ${cpu_upper_threshold},
	"name_prefix": "${name_prefix}",
	"autoscaler_api": "${deployment_name}.${system_domain}",

	"performance": {
		"app_count": ${performance_app_count},
		"app_percentage_to_scale": ${performance_app_percentage_to_scale},
		"setup_workers": ${performance_setup_workers},
		"update_existing_org_quota": ${performance_update_existing_org_quota}
	}
}
EOF
}

write_app_config \
	"${ACCEPTANCE_CONFIG_PATH}" \
	"${use_existing_organization}" "${use_existing_space}" "${existing_organization}" "${existing_space}" \
	"${AUTOSCALER_ORG_MANAGER_USER:-}" "${AUTOSCALER_ORG_MANAGER_PASSWORD:-}" \
	"${AUTOSCALER_OTHER_USER:-}" "${AUTOSCALER_OTHER_USER_PASSWORD:-}"
