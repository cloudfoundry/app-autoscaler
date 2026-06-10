#!/bin/bash
# shellcheck disable=SC2086
set -euo pipefail

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
source "${script_dir}/utils.source.sh"

MTAR_PATH="${MTAR_PATH:-$(find app-autoscaler-mtar -name 'app-autoscaler-release-*.mtar' 2>/dev/null | head -1)}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-app-autoscaler}"

VALID_ORG="${VALID_ORG:-SAP_autoscaler_tests_OSS}"

function generate_secrets() {
  credhub generate --no-overwrite \
    -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsgateway_health_password" \
    --length 16 -t password > /dev/null
}

function build_extension_file() {
  local ext_file
  ext_file="$(mktemp)"

  local health_password syslog_json syslog_ca syslog_cert syslog_key valid_org_guid

  health_password="$(credhub get -n "/bosh-autoscaler/${DEPLOYMENT_NAME}/autoscaler_metricsgateway_health_password" -q)"
  syslog_json="$(credhub get -n /bosh-autoscaler/cf/syslog_agent_log_cache_tls --output-json)"
  syslog_ca="$(echo "${syslog_json}"   | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['value']['ca'].rstrip())")"
  syslog_cert="$(echo "${syslog_json}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['value']['certificate'].rstrip())")"
  syslog_key="$(echo "${syslog_json}"  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['value']['private_key'].rstrip())")"

  valid_org_guid="$(cf org "${VALID_ORG}" --guid)"

  local syslog_ca_inline syslog_cert_inline syslog_key_inline
  syslog_ca_inline="$(echo "${syslog_ca}"   | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')"
  syslog_cert_inline="$(echo "${syslog_cert}" | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')"
  syslog_key_inline="$(echo "${syslog_key}"  | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')"

  cat > "${ext_file}" << EOF
ID: metricsgateway-deploy
extends: com.github.cloudfoundry.app-autoscaler-release
version: 1.0.0
_schema-version: 3.3.0

modules:
  - name: metricsgateway
    parameters:
      instances: 1

resources:
  - name: metricsgateway-config
    parameters:
      config:
        metricsgateway-config:
          cf_server:
            xfcc:
              valid_org_guid: "${valid_org_guid}"
          health:
            basic_auth:
              password: "${health_password}"
          logging:
            level: info
          syslog:
            server_address: log-cache.service.cf.internal
            port: 6067

  - name: syslog-client
    parameters:
      config:
        client_cert: "${syslog_cert_inline}"
        client_key: "${syslog_key_inline}"
        server_ca: "${syslog_ca_inline}"
EOF

  echo "${ext_file}"
}

function setup_security_group() {
  local sg_file
  sg_file="$(dirname "${BASH_SOURCE[0]}")/../security-groups/metricsgateway.json"
  log "Binding metricsgateway security group to org '${cf_org}' space '${cf_space}'"
  cf create-security-group metricsgateway "${sg_file}" || true
  cf update-security-group metricsgateway "${sg_file}"
  cf bind-security-group metricsgateway "${cf_org}" --space "${cf_space}"
}

function deploy_metricsgateway() {
  local ext_file
  ext_file="$(build_extension_file)"

  log "Deploying metricsgateway from ${MTAR_PATH}"
  cf deploy "${MTAR_PATH}" \
    -e "${ext_file}" \
    -m metricsgateway \
    --version-rule ALL \
    -f

  rm -f "${ext_file}"
}

load_bbl_vars
cf_login "${system_domain}"
cf target -o "${cf_org}" -s "${cf_space}"
generate_secrets
deploy_metricsgateway
setup_security_group

log "metricsgateway deployed:"
cf app metricsgateway
