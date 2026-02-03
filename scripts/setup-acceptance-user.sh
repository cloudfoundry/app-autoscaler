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
#   ./setup-acceptance-user.sh
#
# Example:
#   export DEPLOYMENT_NAME=autoscaler-mta-922
#   export AUTOSCALER_TEST_USER=autoscaler-test-user-123
#   ./setup-acceptance-user.sh

script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${script_dir}/vars.source.sh"

SERVICE_NAME="${deployment_name}"

echo "========================================="
echo "Setting up acceptance test user"
echo "========================================="
echo "Organization: ${AUTOSCALER_ORG}"
echo "Username: ${AUTOSCALER_TEST_USER}"
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
if ! cf org "${AUTOSCALER_ORG}" &> /dev/null; then
  echo "ERROR: Organization '${AUTOSCALER_ORG}' does not exist."
  echo "Please create it first with: cf create-org ${AUTOSCALER_ORG}"
  exit 1
fi
echo "✓ Organization exists"
echo ""

# Create user (idempotent - will not fail if user already exists)
echo "Creating user '${AUTOSCALER_TEST_USER}'..."
if cf create-user "${AUTOSCALER_TEST_USER}" "${PASSWORD}" &> /dev/null; then
  echo "✓ User created successfully"
else
  # User might already exist, check if we can authenticate
  echo "User may already exist, checking..."
  if cf auth "${AUTOSCALER_TEST_USER}" "${PASSWORD}" &> /dev/null; then
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
echo "Setting OrgManager role for '${AUTOSCALER_TEST_USER}' in org '${AUTOSCALER_ORG}'..."
if cf set-org-role "${AUTOSCALER_TEST_USER}" "${AUTOSCALER_ORG}" OrgManager; then
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
cf org-users "${AUTOSCALER_ORG}"
echo ""

echo "========================================="
echo "✓ Setup complete!"
echo "========================================="
echo ""
echo "User: ${AUTOSCALER_TEST_USER}"
echo "Password stored in CredHub at: ${CREDHUB_TEST_USER_PASSWORD_PATH}"
echo ""
echo "To retrieve password:"
echo "  credhub get --name='${CREDHUB_TEST_USER_PASSWORD_PATH}'"
echo ""
echo "Note: Service access will be enabled after the service broker is registered"
