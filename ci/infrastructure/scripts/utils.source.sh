
# shellcheck disable=SC2086
bosh_upload_stemcell_opts="${BOSH_UPLOAD_STEMCELL_OPTS:-""}"

function find_or_upload_stemcell_from(){
  local deployment_manifest=$1
  # Determine if we need to upload a stemcell at this point.
  stemcell_os=$(yq eval '.stemcells[] | select(.alias == "default").os' "${deployment_manifest}")
  stemcell_version=$(yq eval '.stemcells[] | select(.alias == "default").version' "${deployment_manifest}")
  stemcell_name="bosh-google-kvm-${stemcell_os}-go_agent"

  if ! bosh stemcells | grep "${stemcell_name}" >/dev/null; then
    URL="https://bosh.io/d/stemcells/${stemcell_name}"
    if [[ "${stemcell_version}" != "latest" ]]; then
	    URL="${URL}?v=${stemcell_version}"
    fi
    wget "$URL" -O stemcell.tgz
    bosh -n upload-stemcell $bosh_upload_stemcell_opts stemcell.tgz
  fi
  return 0
}

# upload release from a bosh.io resource
function upload_release(){
  local release_dir=$1

  pushd "${release_dir}" > /dev/null || exit
    echo "Uploading release from ${release_dir}"
    echo "Listing files in ${release_dir}:"
    log "$(ls -1 ./*.tgz)"
    bosh -n upload-release release.tgz
  popd > /dev/null || exit
  return 0
}

function load_bbl_vars() {
  if [[ -z "${bbl_state_path}" ]]; then
    echo "ERROR: bbl_state_path is not set" >&2
    exit 1
  fi

  director_store="${bbl_state_path}/vars/director-vars-store.yml"
  log "director_store = '${director_store}'"
  if [[ ! -d ${bbl_state_path} ]]; then
    echo "FAILED: Did not find bbl-state folder at ${bbl_state_path}" >&2
    echo "Make sure you have checked out the app-autoscaler-env-bbl-state repository next to the app-autoscaler-release repository to run this target or indicate its location via BBL_STATE_PATH" >&2;
    exit 1;
  fi

  pushd "${bbl_state_path}" > /dev/null || exit
    bbl_env_file="$(mktemp)"
    bbl print-env > "${bbl_env_file}"
    # shellcheck disable=SC1090
    source "${bbl_env_file}"
    rm -f "${bbl_env_file}"
  popd > /dev/null || exit
}

function validate_ops_files() {
  local ops_files=$1

  for ops_file in ${ops_files}; do
    if [[ ! -f "${ops_file}" ]]; then
      echo "ERROR: could not find ops file ${ops_file} in ${PWD}" >&2
      exit 1
    fi
  done
}

function add_var_to_bosh_deploy_opts() {
  local var_name=$1
  local var_value=$2
  bosh_deploy_opts="${bosh_deploy_opts} -v ${var_name}=${var_value}"
  return 0
}

# update_and_bind_security_group <name> <json_file> <bind_mode> [org] [space]
# Updates an existing security group and binds it. Callers are responsible for
# creating the group first (with or without error suppression as appropriate).
# bind_mode: "running" = bind-running-security-group (global)
#            "org"     = bind-security-group <name> <org>
#            "space"   = unbind + bind-security-group <name> <org> --space <space>
function update_and_bind_security_group() {
  local name=$1 json_file=$2 bind_mode=$3 org=${4:-} space=${5:-}

  cf update-security-group "${name}" "${json_file}"

  case "${bind_mode}" in
    running) cf bind-running-security-group "${name}" ;;
    org)     cf bind-security-group "${name}" "${org}" ;;
    space)
      cf unbind-security-group "${name}" "${org}" "${space}"
      cf bind-security-group "${name}" "${org}" --space "${space}"
      ;;
    *) echo "ERROR: unknown bind_mode '${bind_mode}'" >&2; return 1 ;;
  esac
}

function cf_login(){
  local system_domain=$1

  cf api "https://api.${system_domain}" --skip-ssl-validation
  cf auth admin "$(credhub get -n /bosh-autoscaler/cf/cf_admin_password -q)"

  if [[ -n "${CF_ORG}" ]]; then
    cf create-org "${CF_ORG}"
    cf target -o "${CF_ORG}"
  fi

  if [[ -n "${CF_SPACE}" ]]; then
    cf create-space "${CF_SPACE}"
    cf target -s "${CF_SPACE}"
  fi
  return 0
}

