#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
autoscaler_dir="${script_dir}/.."
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

DEST="${DEST:-/tmp/build}"
MTAR_FILENAME="${MTAR_FILENAME:-app-autoscaler-release-v${VERSION}.mtar}"
MODULES="${MODULES:-dbtasks,apiserver,eventgenerator,metricsforwarder,metricsgateway,operator,scheduler,scalingengine,acceptance-tests}"

# Compute extension file path
EXTENSION_FILE="${DEST}/extension-file-${VERSION}.txt"

# Check if mtar file exists
if [ ! -f "${DEST}/${MTAR_FILENAME}" ]; then
	echo "ERROR: MTAR file not found at: ${DEST}/${MTAR_FILENAME}"
	echo "Please run 'make mta-build' first"
	exit 1
fi

# Check if extension file exists
if [ ! -f "${EXTENSION_FILE}" ]; then
	echo "ERROR: Extension file not found at: ${EXTENSION_FILE}"
	echo "Please run 'make build-extension-file' to build the extension file first."
	exit 1
fi

# Navigate to the autoscaler directory
pushd "${autoscaler_dir}" > /dev/null

	bbl_login
	echo "Deploying with extension file: ${EXTENSION_FILE}"
	cf deploy "${DEST}/${MTAR_FILENAME}" --version-rule ALL -f --delete-services -e "${EXTENSION_FILE}" -m "${MODULES}"

popd > /dev/null

# --- Register service broker ---
# Extract broker password from the generated extension file (baked in by build-extension-file.sh)
SERVICE_BROKER_PASSWORD="$(yq '.resources[] | select(.name == "apiserver-config") | .parameters.config."apiserver-config".broker_credentials[0].broker_password' "${EXTENSION_FILE}")"

cf_login

set +e
existing_service_broker="$(cf curl v3/service_brokers | jq --raw-output \
	--arg service_broker_name "${deployment_name:-}" \
	'.resources[] | select(.name == $service_broker_name) | .name')"
set -e

if [[ -n "${existing_service_broker}" ]]; then
	echo "Service Broker ${existing_service_broker} already exists"
	echo " - cleaning up pr"
	pushd "${autoscaler_dir}/acceptance" > /dev/null
		./cleanup.sh
	popd > /dev/null
	echo ' - deleting broker'
	cf delete-service-broker -f "${existing_service_broker}"
fi

echo "Creating service broker ${deployment_name:-} at 'https://${service_broker_name:-}.${system_domain:-}'"
cf create-service-broker "${deployment_name:-}" autoscaler-broker-user "${SERVICE_BROKER_PASSWORD}" "https://${service_broker_name:-}.${system_domain:-}"

cf logout
