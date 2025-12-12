#! /usr/bin/env bash
# shellcheck disable=SC2154,SC1091

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
autoscaler_dir="${script_dir}/.."

build_path="${BUILD_PATH:-$(realpath build)}"
VERSION="${VERSION:-}"
SUM_FILE="${build_path}/artifacts/files.sum.sha256"

if [ -z "${VERSION}" ]; then
	if [ -f "${build_path}/name" ]; then
		VERSION=$(cat "${build_path}/name")
	else
		echo "ERROR: VERSION not set and ${build_path}/name does not exist"
		exit 1
	fi
fi

function create_mtar() {
	set -e
	mkdir -p "${build_path}/artifacts"
	local version=$1
	local build_path=$2
	echo " - creating autoscaler mtar artifact"
	pushd "${autoscaler_dir}" > /dev/null
		make mta-release VERSION="${version}" DEST="${build_path}/artifacts"
	popd > /dev/null
}

function create_tests() {
	set -e
	mkdir -p "${build_path}/artifacts"
	local version=$1
	local build_path=$2
	echo " - creating acceptance test artifact"
	pushd "${autoscaler_dir}" > /dev/null
		make acceptance-release VERSION="${version}" DEST="${build_path}/artifacts"
	popd > /dev/null
}

function create_bindreq_schema() {
	local -r target_dir="${1}"
	echo " - creating bind request schema artifact in ${target_dir}"
	make bind-request-schema TARGET_DIR="${target_dir}"
	return 0
}

echo " - Creating assets for version ${VERSION}..."

pushd "${autoscaler_dir}" > /dev/null
	mkdir -p "${build_path}/artifacts"

	create_bindreq_schema "${build_path}/artifacts"
	create_tests "${VERSION}" "${build_path}"
	create_mtar "${VERSION}" "${build_path}"

	echo " - Generating checksums..."
	sha256sum "${build_path}/artifacts/"* > "${build_path}/artifacts/files.sum.sha256"

	ACCEPTANCE_TEST_TGZ="app-autoscaler-acceptance-tests-v${VERSION}.tgz"
	AUTOSCALER_MTAR="app-autoscaler-release-v${VERSION}.mtar"
	BIND_REQ_SCHEMA='bind-request.schema.json'

	ACCEPTANCE_SHA256=$( grep "${ACCEPTANCE_TEST_TGZ}$" "${SUM_FILE}" | awk '{print $1}' )
	MTAR_SHA256=$( grep "${AUTOSCALER_MTAR}$" "${SUM_FILE}" | awk '{print $1}')
	BR_SCHEMA_SHA256=$( grep "${BIND_REQ_SCHEMA}$" "${SUM_FILE}" | awk '{print $1}')

	echo " - Assets created successfully:"
	echo "   - Acceptance tests: ${ACCEPTANCE_TEST_TGZ} (SHA256: ${ACCEPTANCE_SHA256})"
	echo "   - MTAR: ${AUTOSCALER_MTAR} (SHA256: ${MTAR_SHA256})"
	echo "   - Bind Request Schema: ${BIND_REQ_SCHEMA} (SHA256: ${BR_SCHEMA_SHA256})"
popd > /dev/null

echo " - Completed"
