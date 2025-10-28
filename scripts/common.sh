#! /usr/bin/env bash

# shellcheck disable=SC2154
# This file is intended to be loaded via the `source`-command.

function step(){
	echo "# $1"
}

function retry(){
	max_retries=$1
	shift
	retries=0
	command="$*"
	until [ "${retries}" -eq "${max_retries}" ] || $command; do
		((retries=retries+1))
		echo " - retrying command '${command}' attempt: ${retries}"
	done
	[ "${retries}" -lt "${max_retries}" ] || { echo "ERROR: Command '$*' failed after ${max_retries} attempts"; return 1; }
}

function bosh_login() {
	step "bosh login"
	echo "BBL_STATE_PATH is set to '${BBL_STATE_PATH}'"
	local -r bbl_state_path="${1}"
	if [[ ! -d "${bbl_state_path}" ]]
	then
		echo "⛔ FAILED: Did not find bbl-state folder at ${bbl_state_path}"
		echo 'Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH'
		exit 1;
	fi

	pushd "${bbl_state_path}" > /dev/null
		unset BBL_STATE_DIRECTORY
		eval "$(bbl print-env)"
	popd > /dev/null
}

function cf_login(){
	step 'login to cf'
	cf api "https://api.${system_domain}" --skip-ssl-validation
	cf_admin_password="$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')"
	cf auth admin "$cf_admin_password"
}

function uaa_login(){
  step "login to uaa"
  uaa_client_secret="$(credhub get --quiet --name='/bosh-autoscaler/cf/uaa_admin_client_secret')"
	uaac target "https://uaa.${system_domain}" --skip-ssl-validation
	uaac token client get admin -s "${uaa_client_secret}"
}

function cleanup_acceptance_run(){
	step "cleaning up from acceptance tests"
	pushd "${autoscaler_acceptance_dir}" > /dev/null
		retry 5 ./cleanup.sh
	popd > /dev/null
}

function cleanup_service_broker(){
	step "deleting service broker for deployment '${deployment_name}'"
	SERVICE_BROKER_EXISTS=$(cf service-brokers | grep -c "${service_broker_name}.${system_domain}" || true)
	if [[ $SERVICE_BROKER_EXISTS == 1 ]]; then
		echo "- Service Broker exists, deleting broker '${deployment_name}'"
		retry 3 cf delete-service-broker "${deployment_name}" -f
	fi
}

function cleanup_bosh_deployment(){
	step "deleting bosh deployment '${deployment_name}'"
	retry 3 bosh delete-deployment -d "${deployment_name}" -n
}

function delete_releases(){
	step "deleting releases"
	if [ -n "${deployment_name}" ]
	then
		for release in $(bosh releases | grep -E "${deployment_name}\s+"  | awk '{print $2}')
		do
			 echo "- Deleting bosh release '${release}'"
			 bosh delete-release -n "app-autoscaler/${release}" &
		done
		wait
	fi
}

function cleanup_db(){
	local script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
	step "cleaning up db '${deployment_name}'"
	"${script_dir}/deprovision_db.sh" || echo " - could not deprovision db '${deployment_name}'"
}


function cleanup_bosh(){
	step "cleaning up bosh"
	retry 3 bosh clean-up --all -n
}

function cleanup_credhub(){
	step "cleaning up credhub: '/bosh-autoscaler/${deployment_name}/*'"
	retry 3 credhub delete --path="/bosh-autoscaler/${deployment_name}"
}

function cleanup_apps(){
	step "cleaning up apps"
	local mtar_app
	local space_guid

	cf_target "${autoscaler_org}" "${autoscaler_space}"

	space_guid="$(cf space --guid "${autoscaler_space}")"
	mtar_app="$(curl --header "Authorization: $(cf oauth-token)" "deploy-service.${system_domain}/api/v2/spaces/${space_guid}/mtas"  | jq ". | .[] | .metadata | .id" -r)"

	if [ -n "${mtar_app}" ]; then
		set +e
		cf undeploy "${mtar_app}" -f --delete-service-brokers --delete-service-keys --delete-services --do-not-fail-on-missing-permissions
		set -e
	else
		 echo "No app to undeploy"
	fi

	if cf spaces | grep --quiet --regexp="^${AUTOSCALER_SPACE}$"; then
		cf delete-space -f "${AUTOSCALER_SPACE}"
	fi

	if cf orgs | grep --quiet --regexp="^${AUTOSCALER_ORG}$"
	then
		cf delete-org -f "${AUTOSCALER_ORG}"
	fi
}


function unset_vars() {
	unset PR_NUMBER
	unset DEPLOYMENT_NAME
	unset SYSTEM_DOMAIN
	unset BBL_STATE_PATH
	unset AUTOSCALER_DIR
	unset CI_DIR
	unset SERVICE_NAME
	unset SERVICE_BROKER_NAME
	unset NAME_PREFIX
	unset GINKGO_OPTS
}

function find_or_create_org(){
	step "finding or creating org"
	local org_name="$1"
	if ! cf orgs | grep --quiet --regexp="^${org_name}$"
	then
		cf create-org "${org_name}"
	fi
	echo "targeting org ${org_name}"
	cf target -o "${org_name}"
}

function find_or_create_space(){
	step "finding or creating space"
	local space_name="$1"
	if ! cf spaces | grep --quiet --regexp="^${space_name}$"
	then
		cf create-space "${space_name}"
	fi
	echo "targeting space ${space_name}"
	cf target -s "${space_name}"
}

function cf_target(){
	local org_name="$1"
	local space_name="$2"

	find_or_create_org "${org_name}"
	find_or_create_space "${space_name}"
}

function check_database_exists(){
	local bosh_deployment="${1}"
	local postgres_instance="${2}"
	local db_user="${3}"
	local database_name="${4}"

	local db_exists
	db_exists=$(bosh -d "${bosh_deployment}" ssh "${postgres_instance}" \
		-c "sudo su - vcap -c \"/var/vcap/packages/postgres-16/bin/psql -h 127.0.0.1 -p 5524 -U ${db_user} -d postgres -tAc 'SELECT datname FROM pg_database WHERE datname='\\'${database_name}\\'''\"" \
		2>/dev/null | grep -o "${database_name}" || echo "")

	if [ -z "${db_exists}" ]; then
		return 1
	else
		return 0
	fi
}
