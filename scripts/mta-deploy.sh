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
MODULES="${MODULES:-dbtasks,apiserver,eventgenerator,metricsforwarder,operator,scheduler,scalingengine,acceptance-tests}"

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
