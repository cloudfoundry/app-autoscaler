#! /usr/bin/env bash

set -eu -o pipefail
script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${script_dir}/vars.source.sh"
source "${script_dir}/common.sh"

skip_teardown="${SKIP_TEARDOWN:-false}"
suites="${SUITES:-"api app broker"}"
ginkgo_opts="${GINKGO_OPTS:-}"
nodes="${NODES:-3}"

if [[ ! -d "${BBL_STATE_PATH}" ]]
then
	echo "FAILED: Did not find bbl-state folder at ${BBL_STATE_PATH}"
	echo "Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH";
	exit 1;
fi

if [[ ! -f "${autoscaler_acceptance_dir}/acceptance_config.json" ]]
then
	echo 'FAILED: Did not find file acceptance_config.json.'
	exit 1
fi

suites_to_run=""
for suite in $suites
do
	log "checking suite ${suite}"
	if [[ -d "${suite}" ]]
	then
		log "Adding suite '${suite}' to list"
		suites_to_run="${suites_to_run} ${suite}"
	fi
done

step "running ${suites_to_run}"

#run suites
if [ "${suites_to_run}" != "" ]
then
	# shellcheck disable=SC2086
	SKIP_TEARDOWN="${skip_teardown}" CONFIG="${PWD}/acceptance_config.json" DEBUG='true' ./bin/test -race -nodes="${nodes}" -trace $ginkgo_opts ${suites_to_run}
else
	log 'Nothing to run!'
	exit 1
fi
