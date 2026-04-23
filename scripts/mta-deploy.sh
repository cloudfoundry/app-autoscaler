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
FIPS_EXTENSION_FILE="${DEST}/extension-file-fips-${VERSION}.txt"

# Check if mtar file exists
if [[ ! -f "${DEST}/${MTAR_FILENAME}" ]]; then
	echo "ERROR: MTAR file not found at: ${DEST}/${MTAR_FILENAME}" >&2
	echo "Please run 'make mta-build' first" >&2
	exit 1
fi

# Check if extension file exists
if [[ ! -f "${EXTENSION_FILE}" ]]; then
	echo "ERROR: Extension file not found at: ${EXTENSION_FILE}" >&2
	echo "Please run 'make build-extension-file' to build the extension file first." >&2
	exit 1
fi

# Navigate to the autoscaler directory
pushd "${autoscaler_dir}" > /dev/null

	bbl_login
	make -f metricsforwarder/Makefile set-security-group
	echo "Deploying with extension file: ${EXTENSION_FILE}"
	EXTENSION_FILES="${EXTENSION_FILE}"
	if [[ -f "${FIPS_EXTENSION_FILE}" ]]; then
		echo "FIPS extension file found, including: ${FIPS_EXTENSION_FILE}"
		EXTENSION_FILES="${EXTENSION_FILES},${FIPS_EXTENSION_FILE}"
	fi
	cf deploy "${DEST}/${MTAR_FILENAME}" --version-rule ALL -f --delete-services -e "${EXTENSION_FILES}" -m "${MODULES}"

popd > /dev/null
