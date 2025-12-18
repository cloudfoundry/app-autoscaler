#! /usr/bin/env bash
# shellcheck disable=SC2154,SC1091

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
autoscaler_dir="${script_dir}/.."

# Source common functions
source "${script_dir}/common.sh"

build_path="${BUILD_PATH:-$(realpath build)}"
VERSION="${VERSION:-}"
SUM_FILE="${build_path}/artifacts/files.sum.sha256"

# Determine version if not set
if [ -z "${VERSION}" ]; then
	if [ -f "${build_path}/name" ]; then
		VERSION=$(cat "${build_path}/name")
	else
		echo " - VERSION not set, determining version..."
		mkdir -p "${build_path}"
		determine_next_version
		VERSION=$(cat "${build_path}/name")
	fi
fi

function create_mtar() {
	local version=$1
	local artifact_dir=$2
	echo " - creating autoscaler mtar artifact"
	pushd "${autoscaler_dir}" > /dev/null
		make mta-release VERSION="${version}" DEST="${artifact_dir}"
	popd > /dev/null
}

function create_tests() {
	local version=$1
	local artifact_dir=$2
	echo " - creating acceptance test artifact"
	pushd "${autoscaler_dir}" > /dev/null
		make acceptance-release VERSION="${version}" DEST="${artifact_dir}"
	popd > /dev/null
}

function create_bindreq_schema() {
	local -r artifact_dir="${1}"
	echo " - creating bind request schema artifact in ${artifact_dir}"
	make bind-request-schema TARGET_DIR="${artifact_dir}"
}

echo " - Creating assets for version ${VERSION}..."

pushd "${autoscaler_dir}" > /dev/null
	artifact_dir="${build_path}/artifacts"
	mkdir -p "${artifact_dir}"

	create_bindreq_schema "$artifact_dir"
	create_tests "${VERSION}" "$artifact_dir"
	create_mtar "${VERSION}" "$artifact_dir"

	# Validate artifacts were created
	ACCEPTANCE_TEST_TGZ="app-autoscaler-acceptance-tests-v${VERSION}.tgz"
	AUTOSCALER_MTAR="app-autoscaler-release-v${VERSION}.mtar"
	BIND_REQ_SCHEMA='bind-request.schema.json'

	echo " - Validating artifacts..."
	if [[ ! -f "${artifact_dir}/${ACCEPTANCE_TEST_TGZ}" ]]; then
		echo "ERROR: Acceptance test artifact not found: ${ACCEPTANCE_TEST_TGZ}"
		exit 1
	fi
	if [[ ! -f "${artifact_dir}/${AUTOSCALER_MTAR}" ]]; then
		echo "ERROR: MTAR artifact not found: ${AUTOSCALER_MTAR}"
		exit 1
	fi
	if [[ ! -f "${artifact_dir}/${BIND_REQ_SCHEMA}" ]]; then
		echo "ERROR: Bind request schema not found: ${BIND_REQ_SCHEMA}"
		exit 1
	fi

	echo " - Generating checksums..."
	sha256sum "${artifact_dir}/"* > "${SUM_FILE}"

	ACCEPTANCE_SHA256=$( grep "${ACCEPTANCE_TEST_TGZ}$" "${SUM_FILE}" | awk '{print $1}' )
	MTAR_SHA256=$( grep "${AUTOSCALER_MTAR}$" "${SUM_FILE}" | awk '{print $1}')
	BR_SCHEMA_SHA256=$( grep "${BIND_REQ_SCHEMA}$" "${SUM_FILE}" | awk '{print $1}')

	echo " - Assets created successfully:"
	echo "   - Acceptance tests: ${ACCEPTANCE_TEST_TGZ} (SHA256: ${ACCEPTANCE_SHA256})"
	echo "   - MTAR: ${AUTOSCALER_MTAR} (SHA256: ${MTAR_SHA256})"
	echo "   - Bind Request Schema: ${BIND_REQ_SCHEMA} (SHA256: ${BR_SCHEMA_SHA256})"
popd > /dev/null

echo " - Completed"
