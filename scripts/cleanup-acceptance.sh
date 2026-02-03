#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

function main() {
	bbl_login
	cf_login
	cleanup_acceptance_run
	cleanup_test_user
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
