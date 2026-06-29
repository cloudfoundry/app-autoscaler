#!/usr/bin/env bash
# shellcheck disable=SC2154
#
# This file is intended to be loaded via the source command.

# Enable debug output if DEBUG=true
if [[ "${DEBUG:-false}" == "true" ]]; then
	set -x
fi

function is_pr_deployment() {
	[[ -n "${PR_NUMBER:-}" && "${PR_NUMBER}" != "main" ]]
}

function is_main_deployment() {
	[[ -z "${PR_NUMBER:-}" || "${PR_NUMBER}" == "main" ]]
}

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

function bbl_login() {
	step "bosh login"

	local script_dir
	script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

	local bbl_state_path
	bbl_state_path="${BBL_STATE_PATH:-"${script_dir}/../../app-autoscaler-env-bbl-state/bbl-state"}"

	echo "BBL_STATE_PATH is set to '${bbl_state_path}'"

	if [[ ! -d "${bbl_state_path}" ]]; then
		echo "⛔ FAILED: Did not find bbl-state folder at ${bbl_state_path}"
		echo 'Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository or set BBL_STATE_PATH to its location'
		exit 1
	fi

	unset BBL_STATE_DIRECTORY
	eval "$("${script_dir}/bbl-print-env.sh" "${bbl_state_path}")"
}

function cf_login(){
	step 'login to cf as admin'
	cf api "https://api.${system_domain}" --skip-ssl-validation
	cf_admin_password="$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')"
	cf auth admin "$cf_admin_password"
}

# Login to CF with appropriate credentials for deployment operations
# Uses admin on main branch, org-manager on PR branches
function cf_deployment_login(){
	step 'login to cf for deployment operations'
	if is_main_deployment; then
		cf_login
	else
		if [[ -z "${AUTOSCALER_ORG_MANAGER_USER:-}" ]]; then
			echo "ERROR: AUTOSCALER_ORG_MANAGER_USER is not set" >&2
			return 1
		fi
		if [[ -z "${AUTOSCALER_ORG_MANAGER_PASSWORD:-}" ]]; then
			echo "ERROR: AUTOSCALER_ORG_MANAGER_PASSWORD is not set" >&2
			return 1
		fi
		cf api "https://api.${system_domain}" --skip-ssl-validation
		cf auth "${AUTOSCALER_ORG_MANAGER_USER}" "${AUTOSCALER_ORG_MANAGER_PASSWORD}" --origin uaa
	fi
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

function cleanup_db(){
	local script_dir
	script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
	step "cleaning up db '${deployment_name}'"
	"${script_dir}/deprovision_db.sh" || echo " - could not deprovision db '${deployment_name}'"
}


function cleanup_credhub(){
	step "cleaning up credhub: '/bosh-autoscaler/${deployment_name}/*'"
	retry 3 credhub delete --path="/bosh-autoscaler/${deployment_name}"
}


function cleanup_apps(){
	step "cleaning up apps"
	local mtar_app
	local space_guid

	# Don't use cf_target() here — it errors if the space doesn't exist, and during
	# cleanup the space may already be deleted. Use v3 API for safe existence check.
	local org_guid
	local org_output
	if ! org_output=$(cf org "${autoscaler_org}" --guid 2>&1); then
		if echo "${org_output}" | grep -qi "not found"; then
			echo "Org ${autoscaler_org} does not exist, nothing to clean up"
			return 0
		fi
		echo "WARNING: Failed to query org '${autoscaler_org}': ${org_output}" >&2
		return 0
	fi
	org_guid="${org_output}"
	local spaces_response
	if ! spaces_response=$(cf curl "/v3/spaces?names=${autoscaler_space}&organization_guids=${org_guid}" 2>&1); then
		echo "WARNING: Failed to query spaces API: ${spaces_response}" >&2
		return 0
	fi
	if echo "${spaces_response}" | jq -e '.errors' >/dev/null 2>&1; then
		echo "WARNING: CF API error querying spaces: $(echo "${spaces_response}" | jq -r '.errors[0].detail // "unknown"')" >&2
		return 0
	fi
	space_guid=$(echo "${spaces_response}" | jq -r '.resources[0].guid // empty')
	if [[ -z "${space_guid}" ]]; then
		echo "Space ${autoscaler_space} does not exist, nothing to clean up"
		return 0
	fi
	if ! cf target -o "${autoscaler_org}" 2>/dev/null; then
		echo "WARNING: Could not target org '${autoscaler_org}' — cleanup may be incomplete" >&2
	fi
	local mtas_response
	if mtas_response="$(curl --silent --fail --insecure --header "Authorization: $(cf oauth-token)" "https://deploy-service.${system_domain}/api/v2/spaces/${space_guid}/mtas" 2>/dev/null)"; then
		mtar_app="$(jq -r '.[] | .metadata.id' <<< "${mtas_response}")" || true
	else
		echo "Warning: Failed to fetch MTAs from deploy-service, skipping MTA cleanup"
	fi

	if [ -n "${mtar_app}" ]; then
		set +e
		cf undeploy "${mtar_app}" -f --delete-service-brokers --delete-service-keys --delete-services --do-not-fail-on-missing-permissions
		set -e
	else
		 echo "No app to undeploy"
	fi

	# Purge orphaned service instances from all spaces in the org
	echo "- Purging orphaned service instances from all spaces"
	set +e
	cf spaces 2>/dev/null | tail --lines +4 | awk '{print $1}' | while read -r space_name; do
		if [ -n "${space_name}" ] && [ "${space_name}" != "name" ]; then
			echo "  - Checking space: ${space_name}"
			cf target -s "${space_name}" > /dev/null 2>&1
			# List all service instances (both user-provided and managed)
			cf services 2>/dev/null | grep --invert-match "^Getting services" | grep --invert-match "^name" | tail --lines +3 | awk '{print $1}' | while read -r service_instance; do
				if [ -n "${service_instance}" ] && [ "${service_instance}" != "No" ]; then
					echo "    - Purging service instance: ${service_instance}"
					cf purge-service-instance "${service_instance}" -f 2>&1 | grep --invert-match "FAILED" || true
				fi
			done
		fi
	done
	set -e

	if cf spaces | grep --quiet --regexp="^${AUTOSCALER_SPACE}$"; then
		cf delete-space -f "${AUTOSCALER_SPACE}"
	fi

	# Only delete the org if this deployment owns it (i.e. the org was created for this deployment).
	# When deploying into a shared org (AUTOSCALER_ORG != DEPLOYMENT_NAME), leave the org intact.
	# Context: multiple PRs share SAP_autoscaler_tests_OSS — if one PR's cleanup deletes it,
	# all other running PR deployments lose their space and service broker registrations.
	if [[ "${AUTOSCALER_ORG:-}" == "${DEPLOYMENT_NAME:-}" ]] && cf orgs | grep --quiet --regexp="^${AUTOSCALER_ORG}$"; then
		cf delete-org -f "${AUTOSCALER_ORG}"
	fi
}


function unset_vars() {
	unset PR_NUMBER
	unset DEPLOYMENT_NAME
	unset SYSTEM_DOMAIN
	unset BBL_STATE_PATH
	unset AUTOSCALER_DIR
	unset AUTOSCALER_ORG
	unset AUTOSCALER_SPACE
	unset SERVICE_NAME
	unset SERVICE_BROKER_NAME
	unset NAME_PREFIX
	unset GINKGO_OPTS
}

function cf_target(){
	local org_name="$1"
	local space_name="$2"

	cf target -o "${org_name}" -s "${space_name}"
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

function get_previous_version() {
  local prev_version
  prev_version=${PREV_VERSION:-$(gh release list --limit 1 --exclude-drafts --exclude-pre-releases --json tagName --jq '.[0].tagName' 2>/dev/null)}
  # If no previous version found, default to v15.9.0
  if [ -z "$prev_version" ] || [ "$prev_version" = "null" ]; then
    prev_version="v15.9.0"
  fi
  echo "$prev_version"
}

function determine_next_version(){
  local previous_version
  previous_version=$(get_previous_version)
  echo " - Previous version from GitHub releases: ${previous_version}"
  echo " - Determining next version..."

  # Check if there's an existing draft release
  local draft_version
  draft_version=$(gh release list --limit 10 --json tagName,isDraft --jq '.[] | select(.isDraft == true) | .tagName' | head -1)

  if [ -n "$draft_version" ]; then
    echo " - Found existing draft release: ${draft_version}"
    echo " - Using draft version as next version"
    echo "${draft_version#v}" > "${build_path}/name"
    return
  fi

  # If no draft found, continue with version calculation
  echo " - No draft release found, calculating version from commits..."
  echo " - Previous version: $previous_version"

  # Remove 'v' prefix if present
  local version_number=${previous_version#v}

  # Parse version components
  IFS='.' read -r major minor patch <<< "$version_number"

  # Get commits since last tag
  local commits_since_tag
  commits_since_tag=$(git rev-list "${previous_version}"..HEAD --oneline 2>/dev/null || git rev-list HEAD --oneline)
  local commit_count
  commit_count=$(echo "$commits_since_tag" | wc -l)

  if [ -z "$commits_since_tag" ] || [ "$commit_count" -eq 0 ]; then
    echo " - No commits since last tag, keeping current version"
    echo "$version_number" > "${build_path}/name"
    return
  fi

  # Extract PR numbers from commits (supports both "(#123)" and " #123 " formats)
  local pr_numbers
  pr_numbers=$(echo "$commits_since_tag" | grep -oE '(\(#[0-9]+\)| #[0-9]+ )' | grep -oE '[0-9]+' | sort -u)

  if [ -z "$pr_numbers" ]; then
    echo " - No PR numbers found in commits, incrementing patch version"
    patch=$((patch + 1))
    local new_version="${major}.${minor}.${patch}"
    echo " - Next version: $new_version"
    echo "$new_version" > "${build_path}/name"
    return
  fi

  # Query GitHub API for PR labels and categorize
  local has_breaking=0
  local has_enhancement=0
  local pr_count=0

  echo " - Checking PR labels for version determination..."
  while IFS= read -r pr_num; do
    if [ -n "$pr_num" ]; then
      pr_count=$((pr_count + 1))
      local labels
      labels=$(gh pr view "$pr_num" --json labels --jq '.labels[].name' 2>/dev/null || echo "")

      if echo "$labels" | grep -q "exclude-from-changelog"; then
        echo "   - PR #$pr_num: excluded from changelog"
        continue
      fi

      if echo "$labels" | grep -q "breaking-change"; then
        echo "   - PR #$pr_num: breaking change"
        has_breaking=1
      elif echo "$labels" | grep -q "enhancement"; then
        echo "   - PR #$pr_num: enhancement"
        has_enhancement=1
      fi
    fi
  done <<< "$pr_numbers"

  # Determine version increment based on PR labels
  if [[ "$has_breaking" -eq 1 ]]; then
    major=$((major + 1))
    minor=0
    patch=0
    echo " - Found breaking changes, incrementing major version"
  elif [[ "$has_enhancement" -eq 1 ]]; then
    minor=$((minor + 1))
    patch=0
    echo " - Found enhancements, incrementing minor version"
  else
    patch=$((patch + 1))
    echo " - Found changes, incrementing patch version"
  fi

  local new_version="${major}.${minor}.${patch}"
  echo " - Next version: $new_version"
  echo "$new_version" > "${build_path}/name"
}
