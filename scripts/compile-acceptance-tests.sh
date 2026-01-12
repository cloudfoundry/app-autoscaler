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

compile_suites() {
  for suite in "${SUITES[@]}"; do
      mkdir -p "build/acceptance/${suite}"
      for os in "${OPERATING_SYSTEMS[@]}"; do
        for arch in "${ARCHITECTURES[@]}"; do
          CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" ginkgo build "acceptance/${suite}"

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
  compile_suites
  compile_ginkgo
  cp "acceptance/cleanup.sh" "build/acceptance/cleanup.sh"

  # Copy test app assets
  echo "Copying test app assets..."
  mkdir -p "build/acceptance/assets/app/go_app/build"
  cp -r "acceptance/assets/app/go_app/build/"* "build/acceptance/assets/app/go_app/build/"
  cp -r "acceptance/assets/file/" "build/acceptance/assets/file/"
}

main
