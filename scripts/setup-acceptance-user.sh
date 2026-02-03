#!/usr/bin/env bash

set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"



function get_password_from_credhub() {
    local password_path="${1}"

    log "Generating/retrieving password from CredHub"
    credhub generate --no-overwrite -n "${password_path}" --length 32 -t password > /dev/null
    credhub get --quiet --name="${password_path}"
}

function user_exists_and_can_authenticate() {
    local username="${1}"
    local password="${2}"

    cf auth "${username}" "${password}" &> /dev/null
}

function reauthenticate_as_admin() {
    local admin_password
    admin_password=$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')
    cf auth admin "${admin_password}" &> /dev/null
}

function create_user_if_needed() {
    local username="${1}"
    local password="${2}"

    log "Creating user '${username}'"

    if cf create-user "${username}" "${password}" &> /dev/null; then
        log "✓ User created successfully"
        return 0
    fi

    log "User may already exist, verifying credentials"
    if user_exists_and_can_authenticate "${username}" "${password}"; then
        log "✓ User already exists with valid credentials"
        reauthenticate_as_admin
        return 0
    fi

    echo "ERROR: Failed to create user or authenticate" >&2
    return 1
}

function assign_org_manager_role() {
    local username="${1}"
    local org="${2}"

    log "Setting OrgManager role for '${username}' in org '${org}'"

    if cf set-org-role "${username}" "${org}" OrgManager; then
        log "✓ OrgManager role set successfully"
        return 0
    fi

    echo "ERROR: Failed to set OrgManager role" >&2
    return 1
}

function print_success_summary() {
    local username="${1}"
    local password_path="${2}"

    step "Setup complete!"
    log "User: ${username}"
    log "Password stored in CredHub at: ${password_path}"
    echo ""
    log "To retrieve password:"
    log "  credhub get --name='${password_path}'"
}

function setup_acceptance_user() {
    step "Setting up acceptance test user"
    log "Organization: ${AUTOSCALER_ORG}"
    log "Username: ${AUTOSCALER_TEST_USER}"

    local password
    password=$(get_password_from_credhub "${CREDHUB_TEST_USER_PASSWORD_PATH}")
    log "✓ Password retrieved"

    create_user_if_needed "${AUTOSCALER_TEST_USER}" "${password}" || exit 1
    assign_org_manager_role "${AUTOSCALER_TEST_USER}" "${AUTOSCALER_ORG}" || exit 1

    print_success_summary "${AUTOSCALER_TEST_USER}" "${CREDHUB_TEST_USER_PASSWORD_PATH}"
}

function main() {
		bbl_login
    setup_acceptance_user
}

[ "${BASH_SOURCE[0]}" == "${0}" ] && main "$@"
