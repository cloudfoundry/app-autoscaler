#!/bin/bash
# shellcheck disable=SC2086
set -euo pipefail

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
source "${script_dir}/utils.source.sh"

bosh_deploy_opts=${BOSH_DEPLOY_OPTS:-}
deployment_name="${DEPLOYMENT_NAME:-postgres}"

release_dir="${POSTGRES_DIR:-$(realpath -e ${root_dir}/../postgres-release)}"
repo_dir="${REPO_DIR:-$(realpath -e ${root_dir}/../postgres-repo)}"
deployment_manifest=${DEPLOYMENT_MANIFEST:-"${repo_dir}/templates/postgres.yml"}

release_ops="${repo_dir}/templates/operations"
ops_files=${OPS_FILES:-"${release_ops}/add_static_ips.yml\
                       ${ci_dir}/operations/set-postgres-disk.yml\
                       ${ci_dir}/operations/set-postgres-tls.yml\
                       ${ci_dir}/operations/set-postgres-max-connections.yml\
                       ${ci_dir}/operations/add-multiapps-databases-to-postgres.yml \
                       ${ci_dir}/operations/tag-vms-and-disks.yml \
                       "}



function deploy () {
  local ops_files_to_use=""
  validate_ops_files "${ops_files}"

  for OPS_FILE in ${ops_files}; do
    ops_files_to_use="${ops_files_to_use} -o ${OPS_FILE}"
  done

  credhub set -n /bosh-autoscaler/postgres/postgres_host_or_ip -t value -v "10.0.255.2"

  step "Deploying release with name '${deployment_name}' "
  log "using Ops files: '${ops_files_to_use}'"
  bosh -n -d "${deployment_name}" \
    deploy "${deployment_manifest}" \
    ${ops_files_to_use} \
    ${bosh_deploy_opts}
}

load_bbl_vars
find_or_upload_stemcell_from "${deployment_manifest}"

upload_release "${release_dir}"
deploy

