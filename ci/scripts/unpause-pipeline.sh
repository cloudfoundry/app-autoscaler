#!/usr/bin/env bash
# shellcheck disable=SC2086
#

target="${TARGET:-app-autoscaler}"

function unpause-job(){
  pipeline="$1"
  jobs=$(fly -t "$target" jobs -p "$pipeline" --json | jq ".[] | select(.paused==true) | .name" -r)

  if [[ -z "$jobs" ]]; then
    echo "No paused job in pipeline $pipeline"
    return
  fi

  selected_job=$(gum choose --no-limit $jobs --header "Select jobs to unpause from pipeline $pipeline")


  if [[ -z "$selected_job" ]]; then
    echo "No job selected to unpause"
    return
  fi

  for j in $selected_job; do
    fly -t "$target" unpause-job -j "$pipeline/$j"
  done
}

function unpause-pipeline(){
  payload=$(fly -t "$target" pipelines --json)

  pipelines=$(echo "$payload" | jq ".[] |.name" -r | sort)
  # ignore shellcheck warning
  pipeline=$(gum choose $pipelines "all")

  if [[ "$pipeline" == "all" ]]; then
    for p in $pipelines; do
      fly -t "$target" unpause-pipeline -p "$p"
      unpause-job "$p"
    done
  else
    fly -t "$target" unpause-pipeline -p "$pipeline"
    unpause-job "$pipeline"
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
unpause-pipeline "${@:-}"
