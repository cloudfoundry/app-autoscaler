#!/usr/bin/env bash

set -eu -o pipefail

# This script provisions an OrgManager user for running acceptance tests
# with limited permissions (no Cloud Controller admin required).
# Password is retrieved from CredHub.
#
# Prerequisites:
# - Must be run by a CF admin user
# - Target CF environment must be set (cf target)
# - Organization must already exist
# - DEPLOYMENT_NAME must be set (for CredHub path)
#
# Usage:
#   ./setup-acceptance-user.sh <org-name> <username>
#
# Example:
#   export DEPLOYMENT_NAME=autoscaler-mta-922
#   ./setup-acceptance-user.sh autoscaler-mta-922 autoscaler-test-user

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <org-name> <username>"
  echo ""
  echo "Example:"
  echo "  export DEPLOYMENT_NAME=autoscaler-mta-922"
  echo "  $0 autoscaler-mta-922 autoscaler-test-user"
  exit 1
fi

ORG_NAME="$1"
USERNAME="$2"

script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${script_dir}/vars.source.sh"

SERVICE_NAME="${deployment_name}"

echo "========================================="
echo "Setting up acceptance test user"
echo "========================================="
echo "Organization: ${ORG_NAME}"
echo "Username: ${USERNAME}"
echo "Service: ${SERVICE_NAME}"
echo ""

# Generate or retrieve password from CredHub
echo "Generating/retrieving password from CredHub: ${CREDHUB_TEST_USER_PASSWORD_PATH}"
credhub generate --no-overwrite -n "${CREDHUB_TEST_USER_PASSWORD_PATH}" --length 32 -t password > /dev/null
PASSWORD=$(credhub get --quiet --name="${CREDHUB_TEST_USER_PASSWORD_PATH}")
echo "✓ Password retrieved from CredHub"
echo ""

# Verify org exists
echo "Verifying organization exists..."
if ! cf org "${ORG_NAME}" &> /dev/null; then
  echo "ERROR: Organization '${ORG_NAME}' does not exist."
  echo "Please create it first with: cf create-org ${ORG_NAME}"
  exit 1
fi
echo "✓ Organization exists"
echo ""

# Create user (idempotent - will not fail if user already exists)
echo "Creating user '${USERNAME}'..."
if cf create-user "${USERNAME}" "${PASSWORD}" &> /dev/null; then
  echo "✓ User created successfully"
else
  # User might already exist, check if we can authenticate
  echo "User may already exist, checking..."
  if cf auth "${USERNAME}" "${PASSWORD}" &> /dev/null; then
    echo "✓ User already exists and credentials are valid"
    # Re-authenticate as admin to continue setup
    cf auth admin "$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')" &> /dev/null
  else
    echo "ERROR: Failed to create user or authenticate with provided credentials"
    exit 1
  fi
fi
echo ""

# Set OrgManager role
echo "Setting OrgManager role for '${USERNAME}' in org '${ORG_NAME}'..."
if cf set-org-role "${USERNAME}" "${ORG_NAME}" OrgManager; then
  echo "✓ OrgManager role set successfully"
else
  echo "ERROR: Failed to set OrgManager role"
  exit 1
fi
echo ""

# Verify setup
echo "Verifying setup..."
echo ""
echo "Organization users:"
cf org-users "${ORG_NAME}"
echo ""

echo "========================================="
echo "✓ Setup complete!"
echo "========================================="
echo ""
echo "User: ${USERNAME}"
echo "Password stored in CredHub at: ${CREDHUB_TEST_USER_PASSWORD_PATH}"
echo ""
echo "To retrieve password:"
echo "  credhub get --name='${CREDHUB_TEST_USER_PASSWORD_PATH}'"
echo ""
echo "Note: Service access will be enabled after the service broker is registered"
