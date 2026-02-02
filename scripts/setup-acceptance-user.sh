#!/usr/bin/env bash

set -eu -o pipefail

# This script provisions an OrgManager user for running acceptance tests
# with limited permissions (no Cloud Controller admin required).
#
# Prerequisites:
# - Must be run by a CF admin user
# - Target CF environment must be set (cf target)
# - Organization must already exist
#
# Usage:
#   ./setup-acceptance-user.sh <org-name> <username> <password>
#
# Example:
#   ./setup-acceptance-user.sh autoscaler-mta-922 autoscaler-test-user my-password

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <org-name> <username> <password>"
  echo ""
  echo "Example:"
  echo "  $0 autoscaler-mta-922 autoscaler-test-user my-password"
  exit 1
fi

ORG_NAME="$1"
USERNAME="$2"
PASSWORD="$3"

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

# Switch to OrgManager user to test permissions
echo "Authenticating as OrgManager user '${USERNAME}'..."
if cf auth "${USERNAME}" "${PASSWORD}" &> /dev/null; then
  echo "✓ Authenticated as '${USERNAME}'"
else
  echo "ERROR: Failed to authenticate as OrgManager user"
  exit 1
fi

# Try to enable service access globally as OrgManager
echo "Testing OrgManager permissions: enabling global service access for '${SERVICE_NAME}'..."
if cf enable-service-access "${SERVICE_NAME}"; then
  echo "✓ Service access enabled globally by OrgManager"
else
  echo "ERROR: OrgManager cannot enable service access globally"
  echo "This approach requires a user with admin privileges"
  exit 1
fi
echo ""

# Verify setup
echo "Verifying setup..."
echo ""
echo "Organization users:"
cf org-users "${ORG_NAME}"
echo ""

echo "Service access (global):"
cf service-access -e "${SERVICE_NAME}"
echo ""

echo "========================================="
echo "✓ Setup complete!"
echo "========================================="
echo ""
echo "You can now run acceptance tests with:"
echo ""
echo "export USE_EXISTING_ORGANIZATION=true"
echo "export EXISTING_ORGANIZATION=\"${ORG_NAME}\""
echo "export AUTOSCALER_TEST_USER=\"${USERNAME}\""
echo "export AUTOSCALER_TEST_PASSWORD=\"${PASSWORD}\""
echo "export SKIP_SERVICE_ACCESS_MANAGEMENT=true"
echo ""
echo "make acceptance-tests-config"
echo "make acceptance-tests SUITES=\"api\""
