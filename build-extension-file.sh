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

export SYSTEM_DOMAIN="autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"
export POSTGRES_EXTERNAL_PORT="${PR_NUMBER:-5432}"

export METRICSFORWARDER_HOST="${METRICSFORWARDER_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder"}"
export METRICSFORWARDER_MTLS_HOST="${METRICSFORWARDER_MTLS_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder-mtls"}"
export SCALINGENGINE_HOST="${SCALINGENGINE_HOST:-"${DEPLOYMENT_NAME}-cf-scalingengine"}"
export EVENTGENERATOR_HOST="${EVENTGENERATOR_HOST:-"${DEPLOYMENT_NAME}-cf-eventgenerator"}"
export PUBLICAPISERVER_HOST="${PUBLICAPISERVER_HOST:-"${DEPLOYMENT_NAME}"}"
export SERVICEBROKER_HOST="${SERVICEBROKER_HOST:-"${DEPLOYMENT_NAME}servicebroker"}"

cat << EOF > /tmp/extension-file-secrets.yml.tpl
postgres_ip: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_ip))
metricsforwarder_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsforwarder_health_password))
policy_db_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/database_password))
policy_db_server_ca: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server.ca))
policy_db_client_cert: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server.certificate))
policy_db_client_key: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/postgres_server.private_key))
syslog_client_ca: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.ca))
syslog_client_cert: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.certificate))
syslog_client_key: ((/bosh-autoscaler/cf/syslog_agent_log_cache_tls.private_key))
EOF

credhub interpolate -f "/tmp/extension-file-secrets.yml.tpl" > /tmp/mtar-secrets.yml

export METRICSFORWARDER_APPNAME="${METRICSFORWARDER_APPNAME:-"${DEPLOYMENT_NAME}-metricsforwarder"}"
export METRICSFORWARDER_HEALTH_PASSWORD="$(yq ".metricsforwarder_health_password" /tmp/mtar-secrets.yml)"

export POSTGRES_IP="$(yq ".postgres_ip" /tmp/mtar-secrets.yml)"

export POLICY_DB_PASSWORD="$(yq ".policy_db_password" /tmp/mtar-secrets.yml)"
export POLICY_DB_SERVER_CA="$(yq ".policy_db_server_ca" /tmp/mtar-secrets.yml)"
export POLICY_DB_CLIENT_CERT="$(yq ".policy_db_client_cert" /tmp/mtar-secrets.yml)"
export POLICY_DB_CLIENT_KEY="$(yq ".policy_db_client_key" /tmp/mtar-secrets.yml)"

export SYSLOG_CLIENT_CA="$(yq ".syslog_client_ca" /tmp/mtar-secrets.yml)"
export SYSLOG_CLIENT_CERT="$(yq ".syslog_client_cert" /tmp/mtar-secrets.yml)"
export SYSLOG_CLIENT_KEY="$(yq ".syslog_client_key" /tmp/mtar-secrets.yml)"

if [ -z "${POSTGRES_IP}" ]; then
	POSTGRES_URI="postgres://postgres:${POLICY_DB_PASSWORD}@${DEPLOYMENT_NAME}-postgres.tcp.${SYSTEM_DOMAIN}:${POSTGRES_EXTERNAL_PORT}/autoscaler?application_name=metricsforwarder&sslmode=verify-full"
else
	POSTGRES_URI="postgres://postgres:${POLICY_DB_PASSWORD}@${POSTGRES_IP}:5432/autoscaler?application_name=metricsforwarder&sslmode=verify-ca"

fi

cat <<EOF > "${extension_file_path}"
ID: development
extends: com.github.cloudfoundry.app-autoscaler-release
version: 1.0.0
_schema-version: 3.3.0

modules:
  - name: metricsforwarder
    requires:
    - name: metricsforwarder-config
    - name: database
    - name: syslog-client
    parameters:
      routes:
      - route: ${METRICSFORWARDER_HOST}.\${default-domain}
      - route: ${METRICSFORWARDER_MTLS_HOST}.\${default-domain}


  - name: publicapiserver
    parameters:
      instances: 0
      routes:
      - route: ${PUBLICAPISERVER_HOST}.\${default-domain}
      - route: ${SERVICEBROKER_HOST}.\${default-domain}

resources:
- name: metricsforwarder-config
  parameters:
    config:
      metricsforwarder:
        health:
          basic_auth:
            password: "${METRICSFORWARDER_HEALTH_PASSWORD}"
- name: database
  parameters:
    config:
      uri: "${POSTGRES_URI}"
      client_cert: "${POLICY_DB_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${POLICY_DB_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${POLICY_DB_SERVER_CA//$'\n'/\\n}"
- name: syslog-client
  parameters:
    config:
      client_cert: "${SYSLOG_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${SYSLOG_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${SYSLOG_CLIENT_CA//$'\n'/\\n}"
- name: publicapiserver-config
  parameters:
    config:
      publicapiserver:
        cf:
          skip_ssl_validation: true
        metrics_forwarder:
          metrics_forwarder_url: ${METRICSFORWARDER_HOST}.\${default-domain}
          metrics_forwarder_mtls_url: ${METRICSFORWARDER_MTLS_HOST}.\${default-domain}
        scaling_engine:
          scaling_engine_url: ${SCALINGENGINE_HOST}.\${default-domain}
        event_generator:
          event_generator_url: ${EVENTGENERATOR_HOST}.\${default-domain}
EOF
