#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"
DEST="${DEST:-/tmp/build}"

echo "Building extension file for autoscaler deployment with version ${VERSION}"
extension_file_path="${DEST}/extension-file-${VERSION}.txt"
mkdir -p "${DEST}"

if [ -f "${extension_file_path}" ]; then
  echo "Extension file already exists at: ${extension_file_path}"
  echo "Skipping rebuild. Delete the file to regenerate it."
  exit 0
fi

if [ -z "${DEPLOYMENT_NAME}" ]; then
  echo "DEPLOYMENT_NAME is not set"
  exit 1
fi

bbl_login
cf_deployment_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

export SYSTEM_DOMAIN="autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"
export CPU_LOWER_THRESHOLD="${CPU_LOWER_THRESHOLD:-"100"}"

generate_deployment_secrets() {
  METRICSFORWARDER_HEALTH_PASSWORD="$(openssl rand -base64 12)"
  METRICSGATEWAY_HEALTH_PASSWORD="$(openssl rand -base64 12)"
  OPERATOR_HEALTH_PASSWORD="$(openssl rand -base64 12)"
  EVENTGENERATOR_HEALTH_PASSWORD="$(openssl rand -base64 12)"
  SCALINGENGINE_HEALTH_PASSWORD="$(openssl rand -base64 12)"
  SERVICE_BROKER_PASSWORD_BLUE="$(openssl rand -base64 12)"
  SERVICE_BROKER_PASSWORD="$(openssl rand -base64 12)"
  export METRICSFORWARDER_HEALTH_PASSWORD METRICSGATEWAY_HEALTH_PASSWORD
  export OPERATOR_HEALTH_PASSWORD EVENTGENERATOR_HEALTH_PASSWORD
  export SCALINGENGINE_HEALTH_PASSWORD SERVICE_BROKER_PASSWORD_BLUE SERVICE_BROKER_PASSWORD
}

load_secrets() {
  local secrets_file="$1"
  # Map YAML keys → shell variable names, emitting `export VAR=value` lines
  local exports
  exports="$(yq '
    "export CF_ADMIN_PASSWORD="                       + (.cf_admin_password                       | @sh),
    "export POSTGRES_IP="                             + (.postgres_ip                             | @sh),
    "export DATABASE_DB_USERNAME="                    + (.database_username                       | @sh),
    "export DATABASE_DB_PASSWORD="                    + (.database_password                       | @sh),
    "export DATABASE_DB_SERVER_CA="                   + (.database_server_ca                      | @sh),
    "export DATABASE_DB_CLIENT_CERT="                 + (.database_client_cert                    | @sh),
    "export DATABASE_DB_CLIENT_KEY="                  + (.database_client_key                     | @sh),
    "export SYSLOG_CLIENT_CA="                        + (.syslog_client_ca                        | @sh),
    "export SYSLOG_CLIENT_CERT="                      + (.syslog_client_cert                      | @sh),
    "export SYSLOG_CLIENT_KEY="                       + (.syslog_client_key                       | @sh)
  ' "${secrets_file}")"
  eval "${exports}"
  return
}

# PEM certs contain real newlines; escape them to \n for inline YAML embedding
escape_newlines() { printf '%s' "${1//$'\n'/\\n}"; return; }

generate_deployment_secrets

cat << EOF > /tmp/extension-file-secrets.yml.tpl
postgres_ip: ((/bosh-autoscaler/postgres/postgres_host_or_ip))

eventgenerator_log_cache_uaa_client_id: eventgenerator_log_cache
eventgenerator_log_cache_uaa_client_secret: ((/bosh-autoscaler/cf/uaa_clients_eventgenerator_log_cache_secret))

syslog_client_ca: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.ca))
syslog_client_cert: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.certificate))
syslog_client_key: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.private_key))

database_username: pgadmin
database_password: ((/bosh-autoscaler/postgres/pgadmin_database_password))
database_server_ca: ((/bosh-autoscaler/postgres/postgres_server.ca))
database_client_cert: ((/bosh-autoscaler/postgres/postgres_server.certificate))
database_client_key: ((/bosh-autoscaler/postgres/postgres_server.private_key))

cf_admin_password: ((/bosh-autoscaler/cf/cf_admin_password))
EOF

credhub interpolate -f "/tmp/extension-file-secrets.yml.tpl" > /tmp/mtar-secrets.yml
load_secrets /tmp/mtar-secrets.yml

# --- API server & broker ---
export APISERVER_HOST="${APISERVER_HOST:-"${DEPLOYMENT_NAME}"}"
export APISERVER_INSTANCES="${APISERVER_INSTANCES:-2}"
export SERVICEBROKER_HOST="${SERVICEBROKER_HOST:-"${DEPLOYMENT_NAME}servicebroker"}"

# --- CF credentials for components ---
# PR deployments use password grant with org-manager user.
# Main deployments use client_credentials (configured in the template defaults).
if is_pr_deployment; then
  if [[ -z "${AUTOSCALER_ORG_MANAGER_PASSWORD:-}" ]]; then
    echo "ERROR: AUTOSCALER_ORG_MANAGER_PASSWORD is required for component CF credentials" >&2
    exit 1
  fi
  for component in EVENTGENERATOR SCALINGENGINE OPERATOR; do
    export "${component}_CF_GRANT_TYPE=password"
    export "${component}_CF_CLIENT_ID=cf"
    export "${component}_CF_SECRET="
    export "${component}_CF_USERNAME=${AUTOSCALER_ORG_MANAGER_USER}"
    export "${component}_CF_PASSWORD=${AUTOSCALER_ORG_MANAGER_PASSWORD}"
  done
else
  for component in EVENTGENERATOR SCALINGENGINE OPERATOR; do
    export "${component}_CF_GRANT_TYPE=client_credentials"
    export "${component}_CF_CLIENT_ID=autoscaler_client_id"
    export "${component}_CF_SECRET=autoscaler_client_secret"
    export "${component}_CF_USERNAME="
    export "${component}_CF_PASSWORD="
  done
fi

# --- Event generator ---
export EVENTGENERATOR_CF_HOST="${EVENTGENERATOR_CF_HOST:-"${DEPLOYMENT_NAME}-cf-eventgenerator"}"
export EVENTGENERATOR_HOST="${EVENTGENERATOR_HOST:-"${DEPLOYMENT_NAME}-eventgenerator"}"
export EVENTGENERATOR_INSTANCES="${EVENTGENERATOR_INSTANCES:-2}"

# --- Metrics forwarder ---
export METRICSFORWARDER_HOST="${METRICSFORWARDER_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder"}"
export METRICSFORWARDER_MTLS_HOST="${METRICSFORWARDER_MTLS_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder-mtls"}"
export METRICSFORWARDER_INSTANCES="${METRICSFORWARDER_INSTANCES:-2}"

# --- Metrics gateway ---
export USE_METRICSGATEWAY="${USE_METRICSGATEWAY:-true}"
export METRICSGATEWAY_HOST="${METRICSGATEWAY_HOST:-"${DEPLOYMENT_NAME}-metricsgateway"}"
export METRICSGATEWAY_INSTANCES="${METRICSGATEWAY_INSTANCES:-2}"
export METRICSFORWARDER_METRICS_GATEWAY_URL="${METRICSFORWARDER_METRICS_GATEWAY_URL:-}"
AUTOSCALER_ORG_GUID="$(cf org "${AUTOSCALER_ORG}" --guid)"
export AUTOSCALER_ORG_GUID

# --- Scaling engine ---
export SCALINGENGINE_CF_HOST="${SCALINGENGINE_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scalingengine"}"
export SCALINGENGINE_HOST="${SCALINGENGINE_HOST:-"${DEPLOYMENT_NAME}-scalingengine"}"
export SCALINGENGINE_INSTANCES="${SCALINGENGINE_INSTANCES:-2}"

# --- Scheduler ---
export SCHEDULER_HOST="${SCHEDULER_HOST:-"${DEPLOYMENT_NAME}-scheduler"}"
export SCHEDULER_CF_HOST="${SCHEDULER_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scheduler"}"
export SCHEDULER_INSTANCES="${SCHEDULER_INSTANCES:-2}"

# --- Operator ---
export OPERATOR_HOST="${OPERATOR_HOST:-"${DEPLOYMENT_NAME}-operator"}"
export OPERATOR_INSTANCES="${OPERATOR_INSTANCES:-2}"

# --- Database ---
# Port 5524 is the bosh-deployed postgres proxy port (not the default 5432)
export POSTGRES_URI="postgres://${DATABASE_DB_USERNAME}:${DATABASE_DB_PASSWORD}@${POSTGRES_IP}:5524/${DEPLOYMENT_NAME}?sslmode=verify-ca"
DATABASE_DB_CLIENT_CERT="$(escape_newlines "${DATABASE_DB_CLIENT_CERT}")"; export DATABASE_DB_CLIENT_CERT
DATABASE_DB_CLIENT_KEY="$(escape_newlines "${DATABASE_DB_CLIENT_KEY}")";   export DATABASE_DB_CLIENT_KEY
DATABASE_DB_SERVER_CA="$(escape_newlines "${DATABASE_DB_SERVER_CA}")";     export DATABASE_DB_SERVER_CA

# --- Syslog client ---
SYSLOG_CLIENT_CERT="$(escape_newlines "${SYSLOG_CLIENT_CERT}")"; export SYSLOG_CLIENT_CERT
SYSLOG_CLIENT_KEY="$(escape_newlines "${SYSLOG_CLIENT_KEY}")";   export SYSLOG_CLIENT_KEY
SYSLOG_CLIENT_CA="$(escape_newlines "${SYSLOG_CLIENT_CA}")";     export SYSLOG_CLIENT_CA

# --- Acceptance tests ---
export SKIP_SSL_VALIDATION="${SKIP_SSL_VALIDATION:-true}"
export NAME_PREFIX="${NAME_PREFIX:-ASATS}"
export CPU_UPPER_THRESHOLD="${CPU_UPPER_THRESHOLD:-100}"
export PERFORMANCE_APP_COUNT="${PERFORMANCE_APP_COUNT:-100}"
export PERFORMANCE_APP_PERCENTAGE_TO_SCALE="${PERFORMANCE_APP_PERCENTAGE_TO_SCALE:-30}"
export PERFORMANCE_SETUP_WORKERS="${PERFORMANCE_SETUP_WORKERS:-50}"
export PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA="${PERFORMANCE_UPDATE_EXISTING_ORG_QUOTA:-true}"
export USE_EXISTING_ORGANIZATION="${USE_EXISTING_ORGANIZATION:-true}"
export EXISTING_ORGANIZATION="${EXISTING_ORGANIZATION:-${AUTOSCALER_ORG}}"
export SKIP_SERVICE_ACCESS_MANAGEMENT="${SKIP_SERVICE_ACCESS_MANAGEMENT:-true}"
export USE_EXISTING_USER="${USE_EXISTING_USER:-true}"
export EXISTING_USER="${EXISTING_USER:-${AUTOSCALER_ORG_MANAGER_USER}}"
export EXISTING_USER_PASSWORD="${EXISTING_USER_PASSWORD:-${AUTOSCALER_ORG_MANAGER_PASSWORD}}"
export KEEP_USER_AT_SUITE_END="${KEEP_USER_AT_SUITE_END:-true}"
# When using a shared existing org (not the deployment org), don't reuse a space from the
# deployment org — let cf-test-helpers create a fresh space in the existing org instead.
if [[ "${EXISTING_ORGANIZATION}" != "${AUTOSCALER_ORG}" ]]; then
  export ADD_EXISTING_USER_TO_EXISTING_SPACE="${ADD_EXISTING_USER_TO_EXISTING_SPACE:-false}"
  export USE_EXISTING_SPACE="${USE_EXISTING_SPACE:-false}"
  export EXISTING_SPACE="${EXISTING_SPACE:-}"
else
  export ADD_EXISTING_USER_TO_EXISTING_SPACE="${ADD_EXISTING_USER_TO_EXISTING_SPACE:-true}"
  export USE_EXISTING_SPACE="${USE_EXISTING_SPACE:-true}"
  export EXISTING_SPACE="${EXISTING_SPACE:-${AUTOSCALER_SPACE}}"
fi

# ${default-domain} contains a hyphen so envsubst leaves it untouched (hyphens are invalid in shell variable names)
envsubst < "${script_dir}/extension-file.tpl.yaml" > "${extension_file_path}"

# When not using the metricsgateway, patch the generated file:
# - metricsforwarder gets syslog-client binding (direct syslog path)
# - metricsgateway is set to 0 instances and its config resource is removed
if [[ "${USE_METRICSGATEWAY}" != "true" ]]; then
  yq --inplace '
    (.modules[] | select(.name == "metricsforwarder") | .requires) =
      [{"name": "metricsforwarder-config"}, {"name": "syslog-client"}, {"name": "database"}] |
    (.modules[] | select(.name == "metricsgateway")) =
      {"name": "metricsgateway", "parameters": {"instances": 0}} |
    del(.resources[] | select(.name == "metricsgateway-config")) |
    del(.resources[] | select(.name == "metricsforwarder-config") |
      .parameters.config."metricsforwarder-config".metrics_gateway)
  ' "${extension_file_path}"
fi

echo "MTA Extension file created at: ${extension_file_path}"
