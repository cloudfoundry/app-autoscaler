#!/bin/bash

set -euo pipefail
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
pushd "${script_dir}" > /dev/null
domain=${SYSTEM_DOMAIN:-"autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"}
cf create-org testing
cf target -o testing
cf create-space testing
cf target -s testing
npm install
cf push  -p "${script_dir}" --var domain="${domain}"
cf app test_app --guid
popd > /dev/null