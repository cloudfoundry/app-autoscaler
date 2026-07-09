#! /usr/bin/env bash
set -eu -o pipefail

# 🚸 This script is a helper to quickly delete a large amount of orgs.
#  Call-example from the project's root-folder:
# ./scripts/delete-orgs-with-pattern.sh 'mta-1234-TESTS'
#
# 📃 This file can as well be sourced.

function delete_orgs_matching_pattern {
	local -r pattern="${1}"
	local -r max_parallel_deletions='10'

	local matching_orgs
	matching_orgs="$(cf curl 'v3/organizations?per_page=1000' \
											| jq --raw-output --arg pattern "${pattern}" \
													 '.resources[].name | select(test($pattern))')"
	readonly matching_orgs

	if [[ -z "${matching_orgs}" ]]
	then
		echo "No orgs matched '${pattern}'." >&2
		return 0
	fi

	printf '%s\n%s\n\n%s' \
				 'The following orgs will be deleted:' \
				 "${matching_orgs}" \
				 'Do you want to continue? (JSY/Nn)'
	local do_org_deletion
	read -r do_org_deletion
	readonly do_org_deletion

	if [[ "${do_org_deletion}" =~ ^[JSY]$ ]]
	then
		# Delete all orgs as fast as possible with at most <max-procs> at the same time.
		printf '%s\n' "${matching_orgs}" \
			| xargs --max-procs=${max_parallel_deletions} -I {} \
							cf delete-org '{}' -f
	fi
}

if [[ "$(realpath "${BASH_SOURCE[0]}")" == "$(realpath "${0}")" ]]
then
	if [[ $# -lt 1 ]]
	then
		echo "usage: $(basename "${0}") <pattern>" >&2
		exit 2
	fi
	delete_orgs_matching_pattern "${@}"
fi
