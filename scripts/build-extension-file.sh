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
cf_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

export SYSTEM_DOMAIN="autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"
export CPU_LOWER_THRESHOLD="${CPU_LOWER_THRESHOLD:-"100"}"

generate_deployment_secrets() {
  local prefix="/bosh-autoscaler/${DEPLOYMENT_NAME}"
  credhub generate --no-overwrite -n "${prefix}/autoscaler_metricsforwarder_health_password" --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/autoscaler_metricsgateway_health_password"   --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/autoscaler_operator_health_password"         --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/autoscaler_eventgenerator_health_password"   --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/autoscaler_scalingengine_health_password"    --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/service_broker_password_blue"                --length 16 -t password
  credhub generate --no-overwrite -n "${prefix}/service_broker_password"                     --length 16 -t password
  return
}

load_secrets() {
  local secrets_file="$1"
  # Map YAML keys → shell variable names, emitting `export VAR=value` lines
  local exports
  exports="$(yq '
    "export EVENTGENERATOR_HEALTH_PASSWORD="          + (.eventgenerator_health_password          | @sh),
    "export EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_ID="  + (.eventgenerator_log_cache_uaa_client_id  | @sh),
    "export EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_SECRET=" + (.eventgenerator_log_cache_uaa_client_secret | @sh),
    "export METRICSFORWARDER_HEALTH_PASSWORD="        + (.metricsforwarder_health_password        | @sh),
    "export METRICSGATEWAY_HEALTH_PASSWORD="          + (.metricsgateway_health_password          | @sh),
    "export SCALINGENGINE_HEALTH_PASSWORD="           + (.scalingengine_health_password           | @sh),
    "export OPERATOR_HEALTH_PASSWORD="                + (.operator_health_password                | @sh),
    "export CF_ADMIN_PASSWORD="                       + (.cf_admin_password                       | @sh),
    "export POSTGRES_IP="                             + (.postgres_ip                             | @sh),
    "export DATABASE_DB_USERNAME="                    + (.database_username                       | @sh),
    "export DATABASE_DB_PASSWORD="                    + (.database_password                       | @sh),
    "export DATABASE_DB_SERVER_CA="                   + (.database_server_ca                      | @sh),
    "export DATABASE_DB_CLIENT_CERT="                 + (.database_client_cert                    | @sh),
    "export DATABASE_DB_CLIENT_KEY="                  + (.database_client_key                     | @sh),
    "export SYSLOG_CLIENT_CA="                        + (.syslog_client_ca                        | @sh),
    "export SYSLOG_CLIENT_CERT="                      + (.syslog_client_cert                      | @sh),
    "export SYSLOG_CLIENT_KEY="                       + (.syslog_client_key                       | @sh),
    "export SERVICE_BROKER_PASSWORD_BLUE="            + (.service_broker_password_blue            | @sh),
    "export SERVICE_BROKER_PASSWORD="                 + (.service_broker_password                 | @sh)
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

metricsforwarder_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsforwarder_health_password))
metricsgateway_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsgateway_health_password))
operator_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_operator_health_password))
eventgenerator_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_eventgenerator_health_password))
scalingengine_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_scalingengine_health_password))
service_broker_password_blue: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password_blue))
service_broker_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password))

cf_admin_password: ((/bosh-autoscaler/cf/cf_admin_password))
EOF

credhub interpolate -f "/tmp/extension-file-secrets.yml.tpl" > /tmp/mtar-secrets.yml
load_secrets /tmp/mtar-secrets.yml

# --- API server & broker ---
export APISERVER_HOST="${APISERVER_HOST:-"${DEPLOYMENT_NAME}"}"
export APISERVER_INSTANCES="${APISERVER_INSTANCES:-2}"
export SERVICEBROKER_HOST="${SERVICEBROKER_HOST:-"${DEPLOYMENT_NAME}servicebroker"}"

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
AUTOSCALER_ORG_GUID="$(cf org "${AUTOSCALER_ORG}" --guid)"
export AUTOSCALER_ORG_GUID

# --- Scaling engine ---
export SCALINGENGINE_CF_CLIENT_ID="autoscaler_client_id"
export SCALINGENGINE_CF_CLIENT_SECRET="autoscaler_client_secret"
export SCALINGENGINE_CF_HOST="${SCALINGENGINE_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scalingengine"}"
export SCALINGENGINE_HOST="${SCALINGENGINE_HOST:-"${DEPLOYMENT_NAME}-scalingengine"}"
export SCALINGENGINE_INSTANCES="${SCALINGENGINE_INSTANCES:-2}"

# --- Scheduler ---
export SCHEDULER_HOST="${SCHEDULER_HOST:-"${DEPLOYMENT_NAME}-scheduler"}"
export SCHEDULER_CF_HOST="${SCHEDULER_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scheduler"}"
export SCHEDULER_INSTANCES="${SCHEDULER_INSTANCES:-2}"

# --- Operator ---
export OPERATOR_CF_CLIENT_ID="autoscaler_client_id"
export OPERATOR_CF_CLIENT_SECRET="autoscaler_client_secret"
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
