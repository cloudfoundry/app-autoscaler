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

usage() {
  echo "Usage: $0" >&2
  echo "" >&2
  echo "Environment variables:" >&2
  echo "  BOSH_DEPLOYMENT   - BOSH deployment name (default: postgres)" >&2
  echo "  POSTGRES_INSTANCE - Postgres instance to connect to (default: postgres/0)" >&2
  echo "  DEPLOYMENT_NAME   - Name of the database to drop (default: autoscaler)" >&2
  echo "  DB_USER           - Database user (default: postgres)" >&2
  echo "  BBL_STATE_PATH    - Path to BBL state (optional, for bosh login)" >&2
  echo "" >&2
  exit 1
}

# Check required variables
if [ -z "${DEPLOYMENT_NAME}" ]; then
  echo "Error: DEPLOYMENT_NAME environment variable is required" >&2
  usage
fi

# Login to BOSH if BBL_STATE_PATH is set and valid
if [ -n "${BBL_STATE_PATH:-}" ] && [[ "${BBL_STATE_PATH}" != *"ERR_invalid_state_path"* ]]; then
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
elif [[ "${BBL_STATE_PATH:-}" == *"ERR_invalid_state_path"* ]]; then
  echo "Warning: BBL_STATE_PATH is not set or invalid, skipping bosh login" >&2
  echo "Set BBL_STATE_PATH environment variable if you need to login to bosh" >&2
fi

echo "⚠️  WARNING: This will DROP database '${DEPLOYMENT_NAME}' on Postgres instance ${POSTGRES_INSTANCE} in deployment ${BOSH_DEPLOYMENT}"
echo "⚠️  This operation is DESTRUCTIVE and CANNOT be undone!"

echo "Checking if database exists..."
if ! check_database_exists "${BOSH_DEPLOYMENT}" "${POSTGRES_INSTANCE}" "${DB_USER}" "${DEPLOYMENT_NAME}"; then
  echo "Database '${DEPLOYMENT_NAME}' does not exist. Nothing to deprovision."
else
  echo "Database '${DEPLOYMENT_NAME}' found. Proceeding with deletion..."

  echo "Terminating active connections to database '${DEPLOYMENT_NAME}'..."
  TERMINATE_OUTPUT=$(bosh -d "${BOSH_DEPLOYMENT}" ssh "${POSTGRES_INSTANCE}" \
    -c "sudo su - vcap -c \"PSQL_BIN=\\\$(ls -1d /var/vcap/packages/postgres-*/bin/psql | tail -1) && \\\$PSQL_BIN -h 127.0.0.1 -p 5524 -U ${DB_USER} -d postgres -c 'SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '\\'${DEPLOYMENT_NAME}\\'' AND pid <> pg_backend_pid();'\"" \
    2>&1 || true)

  # Drop the database
  echo "Dropping database '${DEPLOYMENT_NAME}'..."
  DROP_OUTPUT=$(bosh -d "${BOSH_DEPLOYMENT}" ssh "${POSTGRES_INSTANCE}" \
    -c "sudo su - vcap -c \"PSQL_BIN=\\\$(ls -1d /var/vcap/packages/postgres-*/bin/psql | tail -1) && \\\$PSQL_BIN -h 127.0.0.1 -p 5524 -U ${DB_USER} -d postgres -c 'DROP DATABASE \\\"${DEPLOYMENT_NAME}\\\";'\"" \
    2>&1 || true)

  if echo "$DROP_OUTPUT" | grep -qE "DROP DATABASE|does not exist"; then
    echo "Database '${DEPLOYMENT_NAME}' dropped successfully"
  else
    echo "Error dropping database:" >&2
    echo "$DROP_OUTPUT" | grep -vE "Unauthorized use|subject to logging|Connection.*closed" >&2
    exit 1
  fi
fi

echo "Done"
