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

function write_app_config() {
	local -r config_path="$1"
	local -r use_existing_organization="$2"
	local -r use_existing_space="$3"
	local -r existing_org="$4"
	local -r existing_space="$5"

	# MTA deployment exports the hosts and the same credentials used by its
	# health servers. Local/BOSH runs without those hosts keep an empty map.
	# Apiserver is omitted: its public /health is liveness-only, and its
	# separate Prometheus health server is not exposed by the MTA routes.
	jq -n \
		--arg domain "${system_domain}" \
		--arg admin_password "${cf_admin_password}" \
		--arg deployment "${deployment_name}" \
		--arg api_host "${APISERVER_HOST:-${deployment_name}}" \
		--arg prefix "${name_prefix}" \
		--arg existing_org "${existing_org}" \
		--arg existing_space "${existing_space}" \
		--argjson skip_ssl_validation "${skip_ssl_validation}" \
		--argjson use_existing_org "${use_existing_organization}" \
		--argjson use_existing_space "${use_existing_space}" \
		--argjson cpu_upper_threshold "${cpu_upper_threshold}" \
		--argjson fips_mode_expected "${FIPS_MODE_EXPECTED:-false}" \
		--argjson app_count "${performance_app_count}" \
		--argjson percentage_to_scale "${performance_app_percentage_to_scale}" \
		--argjson setup_workers "${performance_setup_workers}" \
		--argjson update_existing_org_quota "${performance_update_existing_org_quota}" '
		def health_endpoint(name; host; username; password):
			if (host // "") == "" then {} else
				{(name): {
					endpoint: ("https://" + host + "." + $domain + "/health"),
					username: username, password: (password // "")
				}}
			end;
		env as $e | {
			api: ("api." + $domain),
			admin_user: "admin",
			admin_password: $admin_password,
			apps_domain: $domain,
			skip_ssl_validation: $skip_ssl_validation,
			use_http: false,
			service_name: $deployment,
			service_plan: "autoscaler-free-plan",
			service_broker: $deployment,
			use_existing_organization: $use_existing_org,
			existing_organization: $existing_org,
			use_existing_space: $use_existing_space,
			existing_space: $existing_space,
			aggregate_interval: 120,
			default_timeout: 120,
			cpu_upper_threshold: $cpu_upper_threshold,
			name_prefix: $prefix,
			autoscaler_api: ($api_host + "." + $domain),
			fips_mode_expected: $fips_mode_expected,
			health_endpoints: (
				health_endpoint("eventgenerator"; $e.EVENTGENERATOR_HOST; "eventgenerator"; $e.EVENTGENERATOR_HEALTH_PASSWORD) +
				health_endpoint("scalingengine"; $e.SCALINGENGINE_HOST; "scalingengine"; $e.SCALINGENGINE_HEALTH_PASSWORD) +
				health_endpoint("metricsforwarder"; $e.METRICSFORWARDER_HOST; "metricsforwarder"; $e.METRICSFORWARDER_HEALTH_PASSWORD) +
				health_endpoint("operator"; $e.OPERATOR_HOST; "operator"; $e.OPERATOR_HEALTH_PASSWORD) +
				health_endpoint("scheduler"; $e.SCHEDULER_CF_HOST; "test-user"; "test-password")
			),
			performance: {
				app_count: $app_count,
				app_percentage_to_scale: $percentage_to_scale,
				setup_workers: $setup_workers,
				update_existing_org_quota: $update_existing_org_quota
			}
		}' > "${config_path}"
}

write_app_config \
	"${ACCEPTANCE_CONFIG_PATH}" \
	"${use_existing_organization}" "${use_existing_space}" "${existing_organization}" "${existing_space}"
