#!/usr/bin/env bash
# shellcheck disable=SC2086
#
#
set -eu -o pipefail

target="${TARGET:-app-autoscaler}"

function delete-pipeline(){
  payload=$(fly --target="$target" pipelines --json)

  pipelines=$(echo "$payload" | jq ".[] |.name" -r | sort)
  # ignore shellcheck warning
  pipeline=$(gum choose $pipelines )

  if [ -n "$pipeline" ]; then
    fly -t "$target" destroy-pipeline -p "$pipeline"
  fi
}

function check-login(){
  if ! fly -t "$target" status;  then
    echo
    echo "fly -t $target login"
    echo
    exit 1
  fi
}

check-login
delete-pipeline "${@:-}"
