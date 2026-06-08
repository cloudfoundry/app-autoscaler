#! /usr/bin/env bash

set -eu -o pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
source "${script_dir}/common.sh"

function main(){
	bosh_login "${BBL_STATE_PATH}"
	bosh upload-stemcell --sha1 "sha256:$(cat gcp-noble-stemcell/sha256)" "$(cat gcp-noble-stemcell/url)"
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
