#!/bin/bash
# shellcheck disable=SC2086
set -euo pipefail

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
source "${script_dir}/utils.source.sh"

function create_postgres_service() {
  postgres_username="pgadmin"
  postgres_database_name="multiapps_controller"
  postgres_hostname="$(credhub get -n /bosh-autoscaler/postgres/postgres_host_or_ip -q)"
  postgres_password="$(credhub get -n /bosh-autoscaler/postgres/pgadmin_database_password -q)"

  # delete existing service
  cf cups deploy-service-database -p "{ \"uri\": \"postgres://${postgres_username}:${postgres_password}@${postgres_hostname}:5524/${postgres_database_name}?ssl=false\", \"username\": \"${postgres_username}\", \"password\": \"${postgres_password}\" }" -t postgres
}


function deploy_multiapps_controller() {
  app_name=deploy-service

  mv multiapps-controller-web-war/*.war .
  pushd multiapps-controller-web-manifest

  cf push --no-start -f ./*.yml "${app_name}"
  cf set-env "${app_name}" JBP_CONFIG_TOMCAT '{"tomcat": {"version": "9.+"}}'
  cf start "${app_name}"
  # scale up to be able to handle huge (>1GB) .MTARs
  cf scale -m 4G -k 2G deploy-service -f

  popd
}

function add_postrgres_security_group() {
  postgres_ip="$(credhub get -n /bosh-autoscaler/postgres/postgres_host_or_ip --quiet)"

  security_group_json_path="$(mktemp)"
  cat <<EOF > "${security_group_json_path}"
 [
  {
    "protocol": "tcp",
    "destination": "${postgres_ip}/32",
    "ports": "5524",
    "description": "allow egress to the internal postgres IP"
  }
 ]
EOF

  cf create-security-group multiapps-postgres-security-group "${security_group_json_path}"
  cf update-security-group multiapps-postgres-security-group "${security_group_json_path}"
  cf unbind-security-group multiapps-postgres-security-group ${cf_org} ${cf_space}
  cf bind-security-group multiapps-postgres-security-group ${cf_org} --space ${cf_space}
}

function cleanup_multiapps_controller() {
  cf delete -f multiapps-controller
  cf delete-service -f deploy-service-database
}

load_bbl_vars
cf_login "${system_domain}"
cleanup_multiapps_controller
create_postgres_service
add_postrgres_security_group
deploy_multiapps_controller
