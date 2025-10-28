#!/usr/bin/env bash

set -euo pipefail

function getConfItem(){
  val=$(jq -r ".$1" "${config}")
  if [ "$val" = "null" ]; then return 1; fi
  echo "$val"
}
function not() {
  if [ "$1" = "false" ]; then
    echo "true"
  elif [ "$1" = "true" ]; then
    echo "false"
 else
   return 1
  fi
}

config=${CONFIG:-""}
DELETE_ORG="true"
DELETE_SPACE="true"
DELETE_USER="true"
SERVICE_PREFIX="autoscaler"
if [ -n "${config}" ] && which jq > /dev/null ; then
  DELETE_ORG=$(not "$(getConfItem 'use_existing_organization' || echo false )")
  DELETE_SPACE=$(not "$(getConfItem 'use_existing_space'|| echo false )")
  DELETE_USER=$(not "$(getConfItem 'use_existing_user'|| echo false )")
  SERVICE_PREFIX=$(getConfItem 'prefix' || echo "autoscaler")
  NAME_PREFIX=$(getConfItem 'name_prefix' || echo "ASATS")
fi

function delete_org(){
  local ORG=$1

  if ! cf delete-org -f "$ORG"; then
    cf target -o "$ORG"

    services=$(cf services | grep "${SERVICE_PREFIX}" |  awk '{ print $1}')
    for service in $services; do
      echo "purging service instance ${service}"
      cf purge-service-instance "$service" -f || echo "ERROR: purge-service-instance '$service' failed"
    done

    if ! cf delete-org -f "$ORG"; then
      offerings=$(cf cf service-brokers | grep "${SERVICE_PREFIX}" |  awk '{ print $1}')
      for offering in $offerings; do
       echo "# purging service offering ${offering}"
       cf purge-service-offering "$offering" -f || echo "ERROR: purge-service-offering '$offering' failed"
      done
      cf delete-org -f "$ORG" || echo "ERROR: delete-org '$ORG' failed"
    fi
  fi
  echo " - deleted org $ORG"
}

function delete_space(){
   local org=$1
   local space=$2
   cf target -o "${org}" -s "${space}"
   if ! cf delete-space -f "$space"; then
      cf target -o "$org" -s "${space}"
      SERVICES=$(cf services | grep "${SERVICE_PREFIX}" |  awk 'NR>1 { print $1}')
      for SERVICE in $SERVICES; do
        cf purge-service-instance "$SERVICE" -f || echo "ERROR: purge-service-instance '$SERVICE' failed"
      done
      cf delete-space -f "$space" || echo "ERROR: delete-org '$org' failed"
    fi
    echo " - deleted space $space"
}

name_prefix=${NAME_PREFIX:-"ASATS|ASUP|CUST_MET"}

if [ "${DELETE_ORG}" = "false" ]; then
  if [ "${DELETE_SPACE}" = "true" ]; then
    org="$(getConfItem 'existing_organization')"
    cf target -o "$org"
    spaces=$(cf spaces |  awk 'NR>3{ print $1}' | grep -E "${name_prefix}" || true)
    for space in ${spaces}; do
      delete_space "$org" "$space" &
    done
  fi
else
  ORGS=$(cf orgs |  awk 'NR>3{ print $1}' | grep -E "${name_prefix}" || true)
  echo "# deleting orgs: '${ORGS}'"
  for ORG in $ORGS; do
    # shellcheck disable=SC2181
    delete_org "$ORG"
  done
fi

if [ "${DELETE_USER}" = "true" ]; then
  if [ -n "${name_prefix}" ]
  then
    for user in $(cf curl /v3/users | jq -r '.resources[].username' | grep "${name_prefix}-" )
    do
      echo " - deleting left over user '${user}'"
      cf delete-user -f "$user"
    done
  fi
fi
wait
