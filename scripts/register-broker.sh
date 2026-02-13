#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

bbl_login
cf_org_manager_login
cf_target "${autoscaler_org}" "${autoscaler_space}"

existing_service_broker="$(cf curl v3/service_brokers | jq --raw-output \
	--arg service_broker_name "${deployment_name:-}" \
	'.resources[] | select(.name == $service_broker_name) | .name' || true)"

if [[ -n "${existing_service_broker}" ]]; then
	step "Cleaning up existing broker '${existing_service_broker}'"

	delete_autoscaler_service_instances
	"${script_dir}/../acceptance/cleanup.sh" || echo " - acceptance cleanup had errors (non-fatal)"
	delete_service_broker "${existing_service_broker}"

	echo " - cleanup complete"
fi

echo "Creating service broker ${deployment_name:-} at 'https://${service_broker_name:-}.${system_domain:-}'"

autoscaler_service_broker_password=$(credhub get --quiet --name="/bosh-autoscaler/${deployment_name:-}/service_broker_password")
cf create-service-broker "${deployment_name:-}" autoscaler-broker-user "$autoscaler_service_broker_password" "https://${service_broker_name:-}.${system_domain:-}" --space-scoped

echo "Service broker registered successfully"
cf logout
