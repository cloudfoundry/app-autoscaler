#!/usr/bin/env bash
set -euo pipefail

mkdir -p build/acceptance

readonly SUITES=("api" "app" "broker" "post_upgrade" "pre_upgrade" "run_performance" "setup_performance")
readonly OPERATING_SYSTEMS=("linux" "darwin")
readonly ARCHITECTURES=("amd64" "arm64") # [amd64: intel 64 bit chips]  [arm64: apple silicon chips]

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
  pushd ./acceptance > /dev/null
    for os in "${OPERATING_SYSTEMS[@]}"; do
      for arch in "${ARCHITECTURES[@]}"; do
        binary_name="ginkgo_v2_${os}_${arch}"
        CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build -o "build/ginkgo_v2_${os}_${arch}" github.com/onsi/ginkgo/v2/ginkgo
        chmod +x "build/${binary_name}"
				mv "build/${binary_name}" "../build/acceptance/${binary_name}"
      done
    done
  popd > /dev/null
}

main() {
  compile_suites
  compile_ginkgo
  cp "acceptance/cleanup.sh" "build/acceptance/cleanup.sh"

  # Copy test app assets
  echo "Copying test app assets..."
  mkdir -p "build/acceptance/assets/app/go_app/build"
  cp -r "acceptance/assets/app/go_app/build/"* "build/acceptance/assets/app/go_app/build/"
  cp -r "acceptance/assets/file/"* "build/acceptance/assets/file/"
}

main
