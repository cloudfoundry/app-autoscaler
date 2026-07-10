#!/usr/bin/env bash
#
# merge-go-sbom.sh — merge the Go dependency SBOM into the MTA SBOM produced by
# `mbt build --create-bom` (Piper's mtaBuild step, run with createBOM: true).
#
# Why this exists:
#   `mbt sbom-gen` only generates SBOMs for Java (maven) and Node (npm) modules; it
#   explicitly skips Go modules ("mbt is not support it by now"). All six autoscaler
#   Go services share one root go.mod, so their dependencies are entirely absent from
#   the merged SBOM that mbt writes to sbom-gen/bom-mta.xml. This script regenerates the
#   Go SBOM with cyclonedx-gomod and merges it into that file, so the published SBOM
#   covers both the Java and Go dependency graphs.
#
# It is wired as a Piper `shellExecute` step in the Build stage (see .pipeline/config.yml)
# so it runs after mtaBuild and before the SBOM validation policy reads the file.
#
# Tooling required (all present in the mtaBuild container image):
#   - cyclonedx-gomod   (generate the Go SBOM from the root go.mod)
#   - cyclonedx (CLI)   (merge + validate CycloneDX documents)

set -euo pipefail

SBOM_FILE="sbom-gen/bom-mta.xml"   # hard-coded by Piper's mtaBuild (--sbom-file-path)
GO_SBOM_FILE="$(mktemp -t bom-go-XXXXXX.xml)"
trap 'rm -f "${GO_SBOM_FILE}"' EXIT

if [[ ! -f "${SBOM_FILE}" ]]; then
	echo "ERROR: ${SBOM_FILE} not found. mtaBuild must run with createBOM: true first." >&2
	exit 1
fi

# Read the MTA id and version from mta.yaml so the merged document keeps the same
# top-level identity mbt assigned.
mta_id="$(awk '/^ID:/    {print $2; exit}' mta.yaml)"
mta_version="$(awk '/^version:/ {print $2; exit}' mta.yaml)"

echo "Generating Go SBOM from root go.mod ..."
cyclonedx-gomod mod -output-version 1.4 -licenses -output "${GO_SBOM_FILE}" .

echo "Merging Go SBOM into ${SBOM_FILE} ..."
cyclonedx merge \
	--hierarchical \
	--name "${mta_id}" \
	--version "${mta_version}" \
	--input-files "${SBOM_FILE}" "${GO_SBOM_FILE}" \
	--output-file "${SBOM_FILE}.new" \
	--input-format xml \
	--output-format xml
mv "${SBOM_FILE}.new" "${SBOM_FILE}"

echo "Validating merged SBOM ..."
cyclonedx validate --input-file "${SBOM_FILE}" --input-format xml --fail-on-errors

echo "Merged Go + Java SBOM written to ${SBOM_FILE}"
