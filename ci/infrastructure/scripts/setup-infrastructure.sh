#!/bin/bash
set -euo pipefail

# Custom setup-infrastructure script that wraps bbl plan + bbl up with a
# template patch step in between. This is needed because the network-lb-gcp
# plan patch (Terraform override approach) is broken with BBL v9 / TF 1.4+.

ROOT_DIR="${PWD}"

# shellcheck disable=SC1091
source cf-deployment-concourse-tasks/shared-functions

pushd "bbl-state"
  write_gcp_service_account_key
popd

mkdir -p "bbl-state/${BBL_STATE_DIR}"
pushd "bbl-state/${BBL_STATE_DIR}"
  bbl version

  name_flag=""
  if [ -n "${BBL_ENV_NAME}" ] && [ ! -f bbl-state.json ]; then
    name_flag="--name ${BBL_ENV_NAME}"
  fi

  # Write LB certs
  if [ -f "${BBL_LB_CERT}" ]; then
    bbl_cert_path="${BBL_LB_CERT}"
  else
    echo "${BBL_LB_CERT}" > /tmp/bbl-cert
    bbl_cert_path="/tmp/bbl-cert"
  fi
  if [ -f "${BBL_LB_KEY}" ]; then
    bbl_key_path="${BBL_LB_KEY}"
  else
    echo "${BBL_LB_KEY}" > /tmp/bbl-key
    bbl_key_path="/tmp/bbl-key"
  fi

  lb_flags="--lb-type=cf --lb-cert=${bbl_cert_path} --lb-key=${bbl_key_path} --lb-domain=${LB_DOMAIN}"

  # Step 1: bbl plan (generates bbl-template.tf)
  echo "=== Running bbl plan ==="
  # shellcheck disable=SC2086
  if [ "${DEBUG_MODE}" == "true" ]; then
    bbl plan --debug ${name_flag} ${lb_flags} 2>&1 | tee "${ROOT_DIR}/bbl_plan.log"
  else
    bbl plan --debug ${name_flag} ${lb_flags} > "${ROOT_DIR}/bbl_plan.log" 2>&1
  fi

  # Step 2: Patch bbl-template.tf to use regional network LB
  echo "=== Patching bbl-template.tf for regional network LB ==="
  "${ROOT_DIR}/ci/ci/infrastructure/scripts/patch-bbl-template.sh" terraform

  # Step 3: bbl up (terraform apply + bosh create-env)
  echo "=== Running bbl up ==="
  # shellcheck disable=SC2086
  if [ "${DEBUG_MODE}" == "true" ]; then
    bbl --debug up ${name_flag} ${lb_flags} 2>&1 | tee "${ROOT_DIR}/bbl_up.log"
  else
    bbl --debug up ${name_flag} ${lb_flags} > "${ROOT_DIR}/bbl_up.log" 2>&1
  fi

  echo "=== Setup complete ==="
  bbl outputs | grep -i "router_lb_ip\|system_domain"

  # Clean up terraform plugins (large)
  rm -rf "terraform/.terraform"
popd

# Commit state changes
commit_bbl_state_dir "${ROOT_DIR}" "Update bbl state dir"
