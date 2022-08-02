#!/bin/bash

set -euo pipefail
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
pushd "${script_dir}" > /dev/null

cf create-org testing
cf target -o testing
cf create-space testing
cf target -s testing
npm install
cf push  -p "${script_dir}"
cf app test_app --guid
popd > /dev/null