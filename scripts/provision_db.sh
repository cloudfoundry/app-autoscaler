#!/usr/bin/env bash

set -euo pipefail

echo "Running $0"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
source "${SCRIPT_DIR}/vars.source.sh"
source "${SCRIPT_DIR}/common.sh"

# Required environment variables
BOSH_DEPLOYMENT="${BOSH_DEPLOYMENT:-postgres}"
POSTGRES_INSTANCE="${POSTGRES_INSTANCE:-postgres/0}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-autoscaler}"
DB_USER="${DB_USER:-vcap}"
APP_DB_USER="${APP_DB_USER:-pgadmin}"

usage() {
  echo "Usage: $0" >&2
  echo "" >&2
  echo "Environment variables:" >&2
  echo "  BOSH_DEPLOYMENT   - BOSH deployment name (default: postgres)" >&2
  echo "  POSTGRES_INSTANCE - Postgres instance to connect to (default: postgres/0)" >&2
  echo "  DEPLOYMENT_NAME   - Name of the database to create (default: autoscaler)" >&2
  echo "  DB_USER           - Database user (default: vcap)" >&2
  echo "  APP_DB_USER       - Application database user to grant permissions (default: pgadmin)" >&2
  echo "  BBL_STATE_PATH    - Path to BBL state (optional, for bosh login)" >&2
  echo "" >&2
  exit 1
}

# Check required variables
if [ -z "${DEPLOYMENT_NAME}" ]; then
  echo "Error: DEPLOYMENT_NAME environment variable is required" >&2
  usage
fi

# Login to BOSH if BBL_STATE_PATH is set
if [ -n "${BBL_STATE_PATH:-}" ]; then
  if [[ ! -d "${BBL_STATE_PATH}" ]]; then
    echo "⛔ FAILED: Did not find bbl-state folder at ${BBL_STATE_PATH}" >&2
    echo 'Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH' >&2
    exit 1
  fi
  echo "# bosh login"
  SAVED_PWD="$(pwd)"
  cd "${BBL_STATE_PATH}"
  eval "$(bbl print-env)"
  cd "${SAVED_PWD}"
fi

echo "Provisioning database '${DEPLOYMENT_NAME}' on Postgres instance ${POSTGRES_INSTANCE} in deployment ${BOSH_DEPLOYMENT}"

# Check if database already exists and create if needed
echo "Checking if database exists..."
if check_database_exists "${BOSH_DEPLOYMENT}" "${POSTGRES_INSTANCE}" "${DB_USER}" "${DEPLOYMENT_NAME}"; then
  echo "Database '${DEPLOYMENT_NAME}' already exists"
else
  echo "Creating database '${DEPLOYMENT_NAME}'"
  CREATE_OUTPUT=$(bosh -d "${BOSH_DEPLOYMENT}" ssh "${POSTGRES_INSTANCE}" \
    -c "sudo su - vcap -c \"PSQL_BIN=\\\$(ls -1d /var/vcap/packages/postgres-*/bin/psql | tail -1) && \\\$PSQL_BIN -h 127.0.0.1 -p 5524 -U ${DB_USER} -d postgres -c 'CREATE DATABASE \\\"${DEPLOYMENT_NAME}\\\";'\"" \
    2>&1 || true)

  if echo "$CREATE_OUTPUT" | grep -qE "CREATE DATABASE|already exists"; then
    echo "Database '${DEPLOYMENT_NAME}' created successfully"
  else
    echo "Error creating database:" >&2
    echo "$CREATE_OUTPUT" | grep -vE "Unauthorized use|subject to logging|Connection.*closed" >&2
    exit 1
  fi
fi

# Grant permissions to the application user
echo "Granting permissions to user '${APP_DB_USER}' on database '${DEPLOYMENT_NAME}'"
GRANT_OUTPUT=$(bosh -d "${BOSH_DEPLOYMENT}" ssh "${POSTGRES_INSTANCE}" \
  -c "sudo su - vcap -c \"PSQL_BIN=\\\$(ls -1d /var/vcap/packages/postgres-*/bin/psql | tail -1) && \\\$PSQL_BIN -h 127.0.0.1 -p 5524 -U ${DB_USER} -d ${DEPLOYMENT_NAME} -c 'GRANT ALL PRIVILEGES ON DATABASE \\\"${DEPLOYMENT_NAME}\\\" TO \\\"${APP_DB_USER}\\\"; GRANT ALL PRIVILEGES ON SCHEMA public TO \\\"${APP_DB_USER}\\\"; ALTER SCHEMA public OWNER TO \\\"${APP_DB_USER}\\\";'\"" \
  2>&1 || true)

if echo "$GRANT_OUTPUT" | grep -qE "GRANT|ALTER"; then
  echo "Permissions granted successfully to '${APP_DB_USER}'"
else
  echo "Warning: Could not grant permissions. Output:" >&2
  echo "$GRANT_OUTPUT" | grep -vE "Unauthorized use|subject to logging|Connection.*closed" >&2
  echo "You may need to grant permissions manually" >&2
fi

echo "Done"
