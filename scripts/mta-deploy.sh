#!/usr/bin/env bash
# shellcheck disable=SC2154

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
autoscaler_dir="${script_dir}/.."
source "${script_dir}/vars.source.sh"
source "${script_dir}/common.sh"

DEST="${DEST:-/tmp/build}"
MTAR_FILENAME="${MTAR_FILENAME:-app-autoscaler-release-v${VERSION:-0.0.0-rc.1}.mtar}"
EXTENSION_FILE="${EXTENSION_FILE:-}"
MODULES="${MODULES:-dbtasks,apiserver,eventgenerator,metricsforwarder,operator,scheduler,scalingengine,acceptance-tests}"

# Check if mtar file exists
if [ ! -f "${DEST}/${MTAR_FILENAME}" ]; then
	echo "ERROR: MTAR file not found at: ${DEST}/${MTAR_FILENAME}"
	echo "Please run 'make mta-build' first"
	exit 1
fi

# Check if extension file exists
if [ -z "${EXTENSION_FILE}" ]; then
	echo "ERROR: EXTENSION_FILE environment variable is not set"
	echo "Please ensure 'build-extension-file' target has been run"
	exit 1
fi

if [ ! -f "${EXTENSION_FILE}" ]; then
	echo "ERROR: Extension file not found at: ${EXTENSION_FILE}"
	echo "Please run 'make build-extension-file' first"
	exit 1
fi

# Navigate to the autoscaler directory
pushd "${autoscaler_dir}" > /dev/null

	bbl_login "${BBL_STATE_PATH}"
	make -f metricsforwarder/Makefile set-security-group
	echo "Deploying with extension file: ${EXTENSION_FILE}"
	cf deploy "${DEST}/${MTAR_FILENAME}" --version-rule ALL -f --delete-services -e "${EXTENSION_FILE}" -m "${MODULES}"

popd > /dev/null
