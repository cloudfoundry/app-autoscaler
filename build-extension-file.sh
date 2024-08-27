#!/usr/bin/env bash
# shellcheck disable=SC2155,SC2034,SC2086

set -e

if [ -z "$1" ]; then
  echo "extension file path not provided"
  exit 1
else
  extension_file_path=$1
fi

if [ -z "${DEPLOYMENT_NAME}" ]; then
  echo "DEPLOYMENT_NAME is not set"
  exit 1
fi

if [ -z "${PR_NUMBER}" ]; then
  echo "PR_NUMBER is not set"
  exit 1
fi

export SYSTEM_DOMAIN="autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"
export POSTGRES_ADDRESS="${DEPLOYMENT_NAME}-postgres.tcp.${SYSTEM_DOMAIN}"
export POSTGRES_EXTERNAL_PORT="${PR_NUMBER:-5432}"

export METRICSFORWARDER_HEALTH_PASSWORD="$(credhub get -n /bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsforwarder_health_password --quiet)"
export METRICSFORWARDER_APPNAME="${METRICSFORWARDER_APPNAME:-"${DEPLOYMENT_NAME}-metricsforwarder"}"

export POLICY_DB_PASSWORD="$(credhub get -n /bosh-autoscaler/${DEPLOYMENT_NAME}/database_password --quiet)"
export POLICY_DB_SERVER_CA="$(credhub get -n /bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server --key ca --quiet )"
export POLICY_DB_CLIENT_CERT="$(credhub get -n /bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server --key certificate --quiet)"
export POLICY_DB_CLIENT_KEY="$(credhub get -n /bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server --key private_key --quiet)"

export SYSLOG_CLIENT_CA="$(credhub get -n /bosh-autoscaler/cf/syslog_agent_log_cache_tls --key ca --quiet)"
export SYSLOG_CLIENT_CERT="$(credhub get -n /bosh-autoscaler/cf/syslog_agent_log_cache_tls --key certificate --quiet)"
export SYSLOG_CLIENT_KEY="$(credhub get -n /bosh-autoscaler/cf/syslog_agent_log_cache_tls --key private_key --quiet)"

cat <<EOF > "${extension_file_path}"
ID: development
extends: com.github.cloudfoundry.app-autoscaler-release
version: 1.0.0
_schema-version: 3.3.0

modules:
  - name: metricsforwarder
    parameters:
      routes:
      - route: ${METRICSFORWARDER_APPNAME}.\${default-domain}

resources:
- name: config
  parameters:
    config:
      metricsforwarder:
        health:
          password: "${METRICSFORWARDER_HEALTH_PASSWORD}"
- name: policydb
  parameters:
    config:
      uri: "postgres://postgres:${POLICY_DB_PASSWORD}@${POSTGRES_ADDRESS}:${POSTGRES_EXTERNAL_PORT}/autoscaler?application_name=metricsforwarder&sslmode=verify-full"
      client_cert: "${POLICY_DB_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${POLICY_DB_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${POLICY_DB_SERVER_CA//$'\n'/\\n}"
- name: syslog-client
  parameters:
    config:
      client_cert: "${SYSLOG_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${SYSLOG_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${SYSLOG_CLIENT_CA//$'\n'/\\n}"
EOF
