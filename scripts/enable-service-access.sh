#!/usr/bin/env bash

set -eu -o pipefail

# This script enables service access globally (as admin)
# It must be run AFTER the service broker is registered
#
# Prerequisites:
# - Service broker must be registered
# - Must be logged in as CF admin
#
# Usage:
#   ./enable-service-access.sh <service-name>
#
# Example:
#   ./enable-service-access.sh autoscaler-mta-922

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <service-name>"
  echo ""
  echo "Example:"
  echo "  $0 autoscaler-mta-922"
  exit 1
fi

SERVICE_NAME="$1"

echo "========================================="
echo "Enabling service access (admin)"
echo "========================================="
echo "Service: ${SERVICE_NAME}"
echo ""

# Enable service access globally
echo "Enabling global service access for '${SERVICE_NAME}'..."
if cf enable-service-access "${SERVICE_NAME}"; then
  echo "✓ Service access enabled globally"
else
  echo "ERROR: Failed to enable service access"
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
