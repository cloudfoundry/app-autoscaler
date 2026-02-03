#!/bin/bash
# Replacement for 'bbl print-env' that reads bbl-state files directly
# Usage: eval "$(./bbl-print-env.sh /path/to/bbl-state)"

set -euo pipefail

bbl_state_dir="${1:-}"

if [ -z "${bbl_state_dir}" ] || [ ! -d "${bbl_state_dir}" ]; then
    echo "Error: bbl-state directory not found: ${bbl_state_dir}" >&2
    exit 1
fi

# Read bbl-state.json for BOSH credentials
bbl_state_json="${bbl_state_dir}/bbl-state.json"
if [ ! -f "${bbl_state_json}" ]; then
    echo "Error: bbl-state.json not found in ${bbl_state_dir}" >&2
    exit 1
fi

# Extract BOSH environment variables
bosh_environment=$(jq -r '.bosh.directorAddress' "${bbl_state_json}")
bosh_client=$(jq -r '.bosh.directorUsername' "${bbl_state_json}")
bosh_client_secret=$(jq -r '.bosh.directorPassword' "${bbl_state_json}")
bosh_ca_cert=$(jq -r '.bosh.directorSSLCA' "${bbl_state_json}")

# Read director vars using yq
director_vars_store="${bbl_state_dir}/vars/director-vars-store.yml"
director_vars_file="${bbl_state_dir}/vars/director-vars-file.yml"
jumpbox_vars_store="${bbl_state_dir}/vars/jumpbox-vars-store.yml"

if [ -f "${director_vars_store}" ]; then
    credhub_secret=$(yq -r '.credhub_admin_client_secret' "${director_vars_store}")
    credhub_ca=$(yq -r '.credhub_ca.ca' "${director_vars_store}")
else
    credhub_secret=""
    credhub_ca=""
fi

if [ -f "${director_vars_file}" ]; then
    internal_ip=$(yq -r '.internal_ip' "${director_vars_file}")
    external_ip=$(yq -r '.external_ip' "${director_vars_file}")
else
    internal_ip=""
    external_ip=""
fi

# Extract jumpbox SSH private key using yq
jumpbox_key_file="${bbl_state_dir}/.jumpbox-ssh-key"

if [ -f "${jumpbox_vars_store}" ]; then
    yq -r '.jumpbox_ssh.private_key' "${jumpbox_vars_store}" > "${jumpbox_key_file}"
    chmod 600 "${jumpbox_key_file}"
fi

jumpbox_proxy="ssh+socks5://jumpbox@${external_ip}:22?private-key=${jumpbox_key_file}"

# Debug: Verify jumpbox key exists and has correct permissions
if [ ! -f "${jumpbox_key_file}" ]; then
    echo "Error: Jumpbox key file not found: ${jumpbox_key_file}" >&2
    exit 1
fi
if [ "$(stat -f '%A' "${jumpbox_key_file}" 2>/dev/null || stat -c '%a' "${jumpbox_key_file}" 2>/dev/null)" != "600" ]; then
    echo "Warning: Jumpbox key file has incorrect permissions" >&2
fi

# Combine CredHub CA with Director CA for the full certificate chain
credhub_ca_cert="${credhub_ca}
${bosh_ca_cert}"

# Output environment variables matching real bbl format
# Simple values: no quotes. Multi-line certificates: single quotes.
printf "export BOSH_CLIENT=%s\n" "${bosh_client}"
printf "export CREDHUB_CLIENT=credhub-admin\n"
printf "export CREDHUB_SERVER=https://%s:8844\n" "${internal_ip}"
printf "export CREDHUB_CA_CERT='%s'\n" "${credhub_ca_cert}"
printf "export CREDHUB_PROXY=%s\n" "${jumpbox_proxy}"
printf "export BOSH_ALL_PROXY=%s\n" "${jumpbox_proxy}"
printf "export BOSH_CLIENT_SECRET=%s\n" "${bosh_client_secret}"
printf "export BOSH_ENVIRONMENT=%s\n" "${bosh_environment}"
printf "export BOSH_CA_CERT='%s'\n" "${bosh_ca_cert}"
printf "export CREDHUB_SECRET=%s\n" "${credhub_secret}"
printf "export JUMPBOX_PRIVATE_KEY=%s\n" "${jumpbox_key_file}"
