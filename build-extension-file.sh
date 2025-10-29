#!/usr/bin/env bash
# shellcheck disable=SC2155,SC2034,SC2086

set -e

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# shellcheck source=../../ci/autoscaler/scripts/vars.source.sh
# shellcheck disable=SC1091
source "${script_dir}/scripts/vars.source.sh"
source "${script_dir}/common.sh"

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

bosh_login "${BBL_STATE_PATH}"
cf_login
cf_target "${AUTOSCALER_ORG}" "${AUTOSCALER_SPACE}"

export SYSTEM_DOMAIN="autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"

export CPU_LOWER_THRESHOLD="${CPU_LOWER_THRESHOLD:-"100"}"

credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsforwarder_health_password" --length 16 -t password
credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_operator_health_password" --length 16 -t password
credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_eventgenerator_health_password" --length 16 -t password
credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_scalingengine_health_password" --length 16 -t password
credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password_blue" --length 16 -t password
credhub generate --no-overwrite -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password" --length 16 -t password

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
operator_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_operator_health_password))
eventgenerator_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_eventgenerator_health_password))
scalingengine_health_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_scalingengine_health_password))
service_broker_password_blue: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password_blue))
service_broker_password: ((/bosh-autoscaler/${DEPLOYMENT_NAME}/service_broker_password))
EOF

credhub interpolate -f "/tmp/extension-file-secrets.yml.tpl" > /tmp/mtar-secrets.yml

export APISERVER_HOST="${APISERVER_HOST:-"${DEPLOYMENT_NAME}"}"
export APISERVER_INSTANCES="${APISERVER_INSTANCES:-2}"
export SERVICEBROKER_HOST="${SERVICEBROKER_HOST:-"${DEPLOYMENT_NAME}servicebroker"}"

export EVENTGENERATOR_HEALTH_PASSWORD="$(yq ".eventgenerator_health_password" /tmp/mtar-secrets.yml)"
export EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_ID="$(yq ".eventgenerator_log_cache_uaa_client_id" /tmp/mtar-secrets.yml)"
export EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_SECRET="$(yq ".eventgenerator_log_cache_uaa_client_secret" /tmp/mtar-secrets.yml)"
export EVENTGENERATOR_CF_HOST="${EVENTGENERATOR_CF_HOST:-"${DEPLOYMENT_NAME}-cf-eventgenerator"}"
export EVENTGENERATOR_HOST="${EVENTGENERATOR_HOST:-"${DEPLOYMENT_NAME}-eventgenerator"}"
export EVENTGENERATOR_INSTANCES="${EVENTGENERATOR_INSTANCES:-2}"

export METRICSFORWARDER_UAA_SKIP_SSL_VALIDATION="$(yq ".metricsforwarder_uaa_skip_ssl_validation" /tmp/mtar-secrets.yml)"
export METRICSFORWARDER_APPNAME="${METRICSFORWARDER_APPNAME:-"${DEPLOYMENT_NAME}-metricsforwarder"}"
export METRICSFORWARDER_HEALTH_PASSWORD="$(yq ".metricsforwarder_health_password" /tmp/mtar-secrets.yml)"
export METRICSFORWARDER_HOST="${METRICSFORWARDER_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder"}"
export METRICSFORWARDER_MTLS_HOST="${METRICSFORWARDER_MTLS_HOST:-"${DEPLOYMENT_NAME}-metricsforwarder-mtls"}"
export METRICSFORWARDER_INSTANCES="${METRICSFORWARDER_INSTANCES:-2}"

export SCALINGENGINE_CF_CLIENT_ID="autoscaler_client_id"
export SCALINGENGINE_CF_CLIENT_SECRET="autoscaler_client_secret"
export SCALINGENGINE_HEALTH_PASSWORD="$(yq ".scalingengine_health_password" /tmp/mtar-secrets.yml)"
export SCALINGENGINE_CF_HOST="${SCALINGENGINE_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scalingengine"}"
export SCALINGENGINE_HOST="${SCALINGENGINE_HOST:-"${DEPLOYMENT_NAME}-scalingengine"}"
export SCALINGENGINE_INSTANCES="${SCALINGENGINE_INSTANCES:-2}"

export SCHEDULER_HOST="${SCHEDULER_HOST:-"${DEPLOYMENT_NAME}-scheduler"}"
export SCHEDULER_CF_HOST="${SCHEDULER_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scheduler"}"
export SCHEDULER_INSTANCES="${SCHEDULER_INSTANCES:-2}"

export SCHEDULER_CF_HOST="${SCHEDULER_CF_HOST:-"${DEPLOYMENT_NAME}-cf-scheduler"}"

export OPERATOR_CF_CLIENT_ID="autoscaler_client_id"
export OPERATOR_CF_CLIENT_SECRET="autoscaler_client_secret"
export OPERATOR_HEALTH_PASSWORD="$(yq ".operator_health_password" /tmp/mtar-secrets.yml)"
export OPERATOR_HOST="${OPERATOR_HOST:-"${DEPLOYMENT_NAME}-operator"}"
export OPERATOR_INSTANCES="${OPERATOR_INSTANCES:-2}"

export POSTGRES_IP="$(yq ".postgres_ip" /tmp/mtar-secrets.yml)"

export DATABASE_DB_USERNAME="$(yq ".database_username" /tmp/mtar-secrets.yml)"
export DATABASE_DB_PASSWORD="$(yq ".database_password" /tmp/mtar-secrets.yml)"
export DATABASE_DB_SERVER_CA="$(yq ".database_server_ca" /tmp/mtar-secrets.yml)"
export DATABASE_DB_CLIENT_CERT="$(yq ".database_client_cert" /tmp/mtar-secrets.yml)"
export DATABASE_DB_CLIENT_KEY="$(yq ".database_client_key" /tmp/mtar-secrets.yml)"

export SYSLOG_CLIENT_CA="$(yq ".syslog_client_ca" /tmp/mtar-secrets.yml)"
export SYSLOG_CLIENT_CERT="$(yq ".syslog_client_cert" /tmp/mtar-secrets.yml)"
export SYSLOG_CLIENT_KEY="$(yq ".syslog_client_key" /tmp/mtar-secrets.yml)"

export SERVICE_BROKER_PASSWORD_BLUE="$(yq ".service_broker_password_blue" /tmp/mtar-secrets.yml)"
export SERVICE_BROKER_PASSWORD="$(yq ".service_broker_password" /tmp/mtar-secrets.yml)"

export POSTGRES_URI="postgres://${DATABASE_DB_USERNAME}:${DATABASE_DB_PASSWORD}@${POSTGRES_IP}:5524/${DEPLOYMENT_NAME}?sslmode=verify-ca"

cat <<EOF > "${extension_file_path}"
ID: development
extends: com.github.cloudfoundry.app-autoscaler-release
version: 1.0.0
_schema-version: 3.3.0

modules:
  - name: apiserver
    parameters:
      instances: ${APISERVER_INSTANCES}
      routes:
      - route: ${APISERVER_HOST}.\${default-domain}
      - route: ${SERVICEBROKER_HOST}.\${default-domain}
    requires:
      - name: apiserver-config
      - name: broker-catalog
      - name: database
  - name: eventgenerator
    requires:
    - name: eventgenerator-config
    - name: database
    parameters:
      instances: ${EVENTGENERATOR_INSTANCES}
      routes:
      - route: ${EVENTGENERATOR_CF_HOST}.\${default-domain}
      - route: ${EVENTGENERATOR_HOST}.\${default-domain}
  - name: scalingengine
    requires:
    - name: scalingengine-config
    - name: database
    parameters:
      instances: ${SCALINGENGINE_INSTANCES}
      routes:
      - route: ${SCALINGENGINE_CF_HOST}.\${default-domain}
      - route: ${SCALINGENGINE_HOST}.\${default-domain}
  - name: metricsforwarder
    requires:
    - name: metricsforwarder-config
    - name: syslog-client
    - name: database
    parameters:
      instances: ${METRICSFORWARDER_INSTANCES}
      routes:
      - route: ${METRICSFORWARDER_HOST}.\${default-domain}
      - route: ${METRICSFORWARDER_MTLS_HOST}.\${default-domain}
  - name: operator
    requires:
    - name: operator-config
    - name: database
    parameters:
      instances: ${OPERATOR_INSTANCES}
      routes:
      - route: ${OPERATOR_HOST}.\${default-domain}
  - name: scheduler
    requires:
    - name: scheduler-config
    - name: database
    parameters:
      instances: ${SCHEDULER_INSTANCES}
      routes:
      - route: ${SCHEDULER_HOST}.\${default-domain}
      - route: ${SCHEDULER_CF_HOST}.\${default-domain}

resources:
- name: metricsforwarder-config
  parameters:
    config:
      metricsforwarder-config:
        health:
          basic_auth:
            password: "${METRICSFORWARDER_HEALTH_PASSWORD}"

- name: eventgenerator-config
  parameters:
    config:
      eventgenerator-config:
        metricCollector:
          metric_collector_url: https://log-cache.\${default-domain}
          port: ''
          uaa:
            client_id: "${EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_ID}"
            client_secret: "${EVENTGENERATOR_LOG_CACHE_UAA_CLIENT_SECRET}"
            url: https://uaa.\${default-domain}
            skip_ssl_validation: true
        pool:
          total_instances: ${EVENTGENERATOR_INSTANCES}
        health:
          basic_auth:
            password: "${EVENTGENERATOR_HEALTH_PASSWORD}"
        scalingEngine:
          scaling_engine_url: https://${SCALINGENGINE_CF_HOST}.\${default-domain}

- name: apiserver-config
  parameters:
    config:
      apiserver-config:
        scaling_rules:
          cpu:
            upper_threshold: $CPU_LOWER_THRESHOLD
        cf:
          api: https://api.$SYSTEM_DOMAIN
          grant_type: client_credentials
          client_id: autoscaler_client_id
          secret: autoscaler_client_secret
        scheduler:
          scheduler_url: https://${SCHEDULER_CF_HOST}.\${default-domain}
        metrics_forwarder:
          metrics_forwarder_url: https://${METRICSFORWARDER_HOST}.\${default-domain}
          metrics_forwarder_mtls_url: https://${METRICSFORWARDER_MTLS_HOST}.\${default-domain}
        scaling_engine:
          scaling_engine_url: https://${SCALINGENGINE_CF_HOST}.\${default-domain}
        event_generator:
          event_generator_url: https://${EVENTGENERATOR_CF_HOST}.\${default-domain}
        broker_credentials:
          - broker_username: 'autoscaler-broker-user'
            broker_password: $SERVICE_BROKER_PASSWORD
          - broker_username: 'autoscaler-broker-user-blue'
            broker_password: $SERVICE_BROKER_PASSWORD_BLUE

- name: scheduler-config
  parameters:
    config:
      cfserver:
        healthserver:
          password: "test-password"
          username: "test-user"
      autoscaler:
        scalingengine:
          url: https://${SCALINGENGINE_HOST}.\${default-domain}
- name: operator-config
  parameters:
    config:
      operator-config:
        health:
          basic_auth:
            password: "${OPERATOR_HEALTH_PASSWORD}"
        cf:
          api:  https://api.\${default-domain}
          client_id: ${OPERATOR_CF_CLIENT_ID}
          secret: ${OPERATOR_CF_CLIENT_SECRET}
        scaling_engine:
          scaling_engine_url: https://${SCALINGENGINE_CF_HOST}.\${default-domain}
        scheduler:
          scheduler_url: https://${SCHEDULER_CF_HOST}.\${default-domain}
- name: scalingengine-config
  parameters:
    config:
      scalingengine-config:
        health:
          basic_auth:
            password: "${SCALINGENGINE_HEALTH_PASSWORD}"
        cf:
          api:  https://api.\${default-domain}
          client_id: ${SCALINGENGINE_CF_CLIENT_ID}
          secret: ${SCALINGENGINE_CF_CLIENT_SECRET}

- name: database
  parameters:
    config:
      uri: "${POSTGRES_URI}"
      client_cert: "${DATABASE_DB_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${DATABASE_DB_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${DATABASE_DB_SERVER_CA//$'\n'/\\n}"

- name: syslog-client
  parameters:
    config:
      client_cert: "${SYSLOG_CLIENT_CERT//$'\n'/\\n}"
      client_key: "${SYSLOG_CLIENT_KEY//$'\n'/\\n}"
      server_ca: "${SYSLOG_CLIENT_CA//$'\n'/\\n}"

- name: broker-catalog
  parameters:
    config:
      broker-catalog:
        services:
          - bindable: true
            bindings_retrievable: true
            description: Automatically increase or decrease the number of application instances based on a policy you define.
            id: autoscaler-guid
            instances_retrievable: true
            name: ${DEPLOYMENT_NAME}
            plans:
              - description: This is the free service plan for the Auto-Scaling service.
                id: autoscaler-free-plan-id
                name: autoscaler-free-plan
                plan_updateable: true
              - description: This is the standard service plan for the Auto-Scaling service.
                id: acceptance-standard
                name: acceptance-standard
                plan_updateable: false
            tags:
              - app-autoscaler
EOF
