#!/usr/bin/env bash
# shellcheck disable=SC2154
#
# Creates the autoscaler_client_id UAA client that the autoscaler components
# (API Server, Scaling Engine, Operator, Event Generator) use to authenticate
# with Cloud Foundry API and UAA via OAuth2 client credentials flow.
#
# Required authorities:
# - cloud_controller.read: Query application state and metadata
# - cloud_controller.admin: Scale application instances up/down, sync schedules
# - uaa.resource: Introspect user tokens via UAA /introspect endpoint (API server)
# - routing.routes.{read,write}: Manage application routes
# - routing.router_groups.read: Read router group information
#
# Usage:
#   ./scripts/create-autoscaler-uaa-client.sh

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

# Configuration
UAA_CLIENT_ID="autoscaler_client_id"
UAA_CLIENT_SECRET="${AUTOSCALER_CLIENT_SECRET:-autoscaler_client_secret}"
UAA_AUTHORITIES="cloud_controller.read,cloud_controller.admin,uaa.resource,routing.routes.write,routing.routes.read,routing.router_groups.read"
GRANT_TYPES="client_credentials"

echo "Creating UAA client for App Autoscaler..."
echo "  Client ID: ${UAA_CLIENT_ID}"
echo "  Authorities: ${UAA_AUTHORITIES}"
echo "  Grant Types: ${GRANT_TYPES}"

# Login to BOSH and get UAA admin credentials
bbl_login

# Get UAA admin client secret from credhub
echo ""
echo "Retrieving UAA admin client secret from credhub..."
UAA_ADMIN_SECRET="$(credhub get -n /bosh-autoscaler/cf/uaa_admin_client_secret -q)"

if [ -z "${UAA_ADMIN_SECRET}" ]; then
  echo "ERROR: Failed to retrieve UAA admin secret from credhub"
  exit 1
fi

# Target UAA
echo ""
echo "Targeting UAA at: https://uaa.${SYSTEM_DOMAIN}"
uaa target "https://uaa.${SYSTEM_DOMAIN}" --skip-ssl-validation

# Authenticate as admin
echo ""
echo "Authenticating as admin client..."
uaa get-client-credentials-token admin -s "${UAA_ADMIN_SECRET}"

# Check if client already exists
echo ""
if uaa get-client "${UAA_CLIENT_ID}" &> /dev/null; then
  echo "Client '${UAA_CLIENT_ID}' already exists. Updating..."
  uaa update-client "${UAA_CLIENT_ID}" \
    --client_secret "${UAA_CLIENT_SECRET}" \
    --authorized_grant_types "${GRANT_TYPES}" \
    --authorities "${UAA_AUTHORITIES}"
  echo "✓ Client '${UAA_CLIENT_ID}' updated successfully"
else
  echo "Creating new client '${UAA_CLIENT_ID}'..."
  uaa create-client "${UAA_CLIENT_ID}" \
    --client_secret "${UAA_CLIENT_SECRET}" \
    --authorized_grant_types "${GRANT_TYPES}" \
    --authorities "${UAA_AUTHORITIES}"
  echo "✓ Client '${UAA_CLIENT_ID}' created successfully"
fi

# Verify the client was created/updated
echo ""
echo "Verifying client configuration..."
uaa get-client "${UAA_CLIENT_ID}"

echo ""
echo "✓ UAA client setup complete!"
echo ""
echo "The Scaling Engine and Operator can now authenticate using:"
echo "  Client ID: ${UAA_CLIENT_ID}"
echo "  Client Secret: ${UAA_CLIENT_SECRET}"
echo "  Grant Type: ${GRANT_TYPES}"
echo "  Authorities: ${UAA_AUTHORITIES}"
