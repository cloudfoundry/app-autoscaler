#!/usr/bin/env bash

set -eu -o pipefail
script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${script_dir}/vars.source.sh"


echo "========================================="
echo "Setting up acceptance test user"
echo "========================================="
echo "Organization: ${AUTOSCALER_ORG}"
echo "Username: ${AUTOSCALER_TEST_USER}"
echo ""

echo "Generating/retrieving password from CredHub: ${CREDHUB_TEST_USER_PASSWORD_PATH}"
credhub generate --no-overwrite -n "${CREDHUB_TEST_USER_PASSWORD_PATH}" --length 32 -t password > /dev/null
PASSWORD=$(credhub get --quiet --name="${CREDHUB_TEST_USER_PASSWORD_PATH}")
echo "✓ Password retrieved from CredHub"
echo ""

echo "Creating user '${AUTOSCALER_TEST_USER}'..."
if cf create-user "${AUTOSCALER_TEST_USER}" "${PASSWORD}" &> /dev/null; then
  echo "✓ User created successfully"
else
  echo "User may already exist, checking..."
  if cf auth "${AUTOSCALER_TEST_USER}" "${PASSWORD}" &> /dev/null; then
    echo "✓ User already exists and credentials are valid"
    cf auth admin "$(credhub get --quiet --name='/bosh-autoscaler/cf/cf_admin_password')" &> /dev/null
  else
    echo "ERROR: Failed to create user or authenticate with provided credentials"
    exit 1
  fi
fi
echo ""

echo "Setting OrgManager role for '${AUTOSCALER_TEST_USER}' in org '${AUTOSCALER_ORG}'..."
if cf set-org-role "${AUTOSCALER_TEST_USER}" "${AUTOSCALER_ORG}" OrgManager; then
  echo "✓ OrgManager role set successfully"
else
  echo "ERROR: Failed to set OrgManager role"
  exit 1
fi
echo ""


echo "========================================="
echo "✓ Setup complete!"
echo "========================================="
echo ""
echo "User: ${AUTOSCALER_TEST_USER}"
echo "Password stored in CredHub at: ${CREDHUB_TEST_USER_PASSWORD_PATH}"
