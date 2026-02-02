#!/usr/bin/env bash

set -eu -o pipefail

# This script enables service access as an OrgManager user
# It must be run AFTER the service broker is registered
#
# Prerequisites:
# - Service broker must be registered
# - User must exist and be logged in as admin
#
# Usage:
#   ./enable-service-access.sh <service-name> <username> <password>
#
# Example:
#   ./enable-service-access.sh autoscaler-mta-922 autoscaler-test-user my-password

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <service-name> <username> <password>"
  echo ""
  echo "Example:"
  echo "  $0 autoscaler-mta-922 autoscaler-test-user my-password"
  exit 1
fi

SERVICE_NAME="$1"
USERNAME="$2"
PASSWORD="$3"

echo "========================================="
echo "Enabling service access as OrgManager"
echo "========================================="
echo "Service: ${SERVICE_NAME}"
echo "Username: ${USERNAME}"
echo ""

# Authenticate as the OrgManager user
echo "Authenticating as '${USERNAME}'..."
if cf auth "${USERNAME}" "${PASSWORD}" &> /dev/null; then
  echo "✓ Authenticated as '${USERNAME}'"
else
  echo "ERROR: Failed to authenticate as OrgManager user"
  exit 1
fi

# Enable service access globally
echo "Enabling global service access for '${SERVICE_NAME}'..."
if cf enable-service-access "${SERVICE_NAME}"; then
  echo "✓ Service access enabled globally"
else
  echo "ERROR: Failed to enable service access"
  echo "OrgManager may not have sufficient permissions"
  exit 1
fi
echo ""

# Verify service access
echo "Verifying service access:"
cf service-access -e "${SERVICE_NAME}"
echo ""

echo "========================================="
echo "✓ Service access enabled!"
echo "========================================="
