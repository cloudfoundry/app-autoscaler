#!/usr/bin/env bash
set -euo pipefail

mkdir -p build/acceptance

# Allow environment variables to override for selective builds
SUITES="${SUITES:-api app broker post_upgrade pre_upgrade run_performance setup_performance}"
OPERATING_SYSTEMS="${TARGET_OS:-linux darwin}"
ARCHITECTURES="${TARGET_ARCH:-amd64 arm64}"

# Convert space-separated strings to arrays
read -ra SUITES <<< "$SUITES"
read -ra OPERATING_SYSTEMS <<< "$OPERATING_SYSTEMS"
read -ra ARCHITECTURES <<< "$ARCHITECTURES"

# Temp dir for the host-native ginkgo tool; cleaned up on exit. Declared at the
# top level (not inside a function) so the EXIT trap can reference it safely even
# under `set -u`, since the trap fires after main() returns and its locals are gone.
GINKGO_TMPDIR=""
cleanup() { [[ -n "${GINKGO_TMPDIR}" ]] && rm -rf "${GINKGO_TMPDIR}"; }
trap cleanup EXIT

# Build a host-native ginkgo tool from the version pinned in acceptance/go.mod
# into the given path. Using `go build` (rather than a pre-installed `ginkgo` on
# PATH) lets this run in any build image that has the go toolchain — e.g. the mbt
# CI image — without ginkgo installed separately. The tool must be built for the
# HOST platform (no GOOS/GOARCH override) so it can execute here; it cross-compiles
# each suite for the target platform via the GOOS/GOARCH env passed to it.
build_host_ginkgo() {
  local out="${1}"
  go build -C acceptance -o "${out}" github.com/onsi/ginkgo/v2/ginkgo
}

compile_suites() {
  local ginkgo_bin="${1}"
  for suite in "${SUITES[@]}"; do
      mkdir -p "build/acceptance/${suite}"
      for os in "${OPERATING_SYSTEMS[@]}"; do
        for arch in "${ARCHITECTURES[@]}"; do
          ( cd acceptance && CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" "${ginkgo_bin}" build "${suite}" )

          # adjust binary name to include os and architecture information
          mv "acceptance/${suite}/${suite}.test" "build/acceptance/${suite}/${suite}_${os}_${arch}.test"
        done
      done
  done
}

compile_ginkgo() {
  for os in "${OPERATING_SYSTEMS[@]}"; do
    for arch in "${ARCHITECTURES[@]}"; do
      binary_name="ginkgo_v2_${os}_${arch}"
      output_path="build/acceptance/${binary_name}"
      CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build -C acceptance -o "../${output_path}" github.com/onsi/ginkgo/v2/ginkgo
      chmod +x "${output_path}"
    done
  done
}

main() {
  # Build the ginkgo tool once for the host platform into a temp dir, then use it
  # to cross-compile the suites. Absolute path so it works after `cd acceptance`.
  local ginkgo_bin
  GINKGO_TMPDIR="$(mktemp -d)"
  ginkgo_bin="${GINKGO_TMPDIR}/ginkgo"
  build_host_ginkgo "${ginkgo_bin}"

  compile_suites "${ginkgo_bin}"
  compile_ginkgo
  cp "acceptance/cleanup.sh" "build/acceptance/cleanup.sh"

  # Copy test app assets
  echo "Copying test app assets..."
  mkdir -p "build/acceptance/assets/app/go_app/build"
  cp -r "acceptance/assets/app/go_app/build/"* "build/acceptance/assets/app/go_app/build/"
  cp -r "acceptance/assets/file/" "build/acceptance/assets/file/"
}

main
