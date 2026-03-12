#! /usr/bin/env bash
# shellcheck disable=SC2154

set -eu -o pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
autoscaler_dir="${script_dir}/.."

# Locally the subsequent shell-check-directives don't not work. However they are
# needed as they are for CI/CD.

# shellcheck source=./scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=./scripts/common.sh
source "${script_dir}/common.sh"

DEST="${DEST:-/tmp/build}"
MTAR_FILENAME="${MTAR_FILENAME:-app-autoscaler-release-v${VERSION}.mtar}"

# Check if mtar file already exists
if [ -f "${DEST}/${MTAR_FILENAME}" ]; then
	echo "⚠️ Existing mtar build found at: ${DEST}/${MTAR_FILENAME}"
	echo "⚠️ Delete the file if you want to recreate it"
	du -h "${DEST}/${MTAR_FILENAME}"
	exit 0
fi

echo "building mtar file for version: ${VERSION}"

# Navigate to the autoscaler directory
pushd "${autoscaler_dir}" > /dev/null
	# Copy template and perform substitutions
	cp mta.tpl.yaml mta.yaml
	sed --in-place "s/MTA_VERSION/${VERSION}/g" mta.yaml

	# Create destination directory
	mkdir --parents "${DEST}"

	# Build the mtar file
	mbt build --target="${DEST}" --mtar="${MTAR_FILENAME}"

	echo "⚠️ The mta build is done. The mtar file is available at: ${DEST}/${MTAR_FILENAME}"
	du --human-readable "${DEST}/${MTAR_FILENAME}"
popd > /dev/null
