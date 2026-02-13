#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

DEST="${DEST:-/tmp/build}"
MTAR_FILENAME="${MTAR_FILENAME:-app-autoscaler-release-v${VERSION}.mtar}"
MODULES="${MODULES:-dbtasks,apiserver,eventgenerator,metricsforwarder,operator,scheduler,scalingengine,acceptance-tests}"
EXTENSION_FILE="${DEST}/extension-file-${VERSION}.txt"

if [ ! -f "${DEST}/${MTAR_FILENAME}" ]; then
	echo "ERROR: MTAR file not found at: ${DEST}/${MTAR_FILENAME}"
	echo "Please run 'make mta-build' first"
	exit 1
fi

if [ ! -f "${EXTENSION_FILE}" ]; then
	echo "ERROR: Extension file not found at: ${EXTENSION_FILE}"
	echo "Please run 'make build-extension-file' to build the extension file first."
	exit 1
fi

bbl_login
cf_org_manager_login
cf_target "${autoscaler_org}" "${autoscaler_space}"
echo "Deploying as user: $(cf target | grep 'user:' | awk '{print $2}')"
echo "Deploying with extension file: ${EXTENSION_FILE}"
cf deploy "${DEST}/${MTAR_FILENAME}" --version-rule ALL -f --delete-services -e "${EXTENSION_FILE}" -m "${MODULES}"
