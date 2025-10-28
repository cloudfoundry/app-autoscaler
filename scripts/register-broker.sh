#! /usr/bin/env bash

set -eu -o pipefail

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"
source "${script_dir}/common.sh"

bosh_login "${BBL_STATE_PATH}"
cf_login

set +e
existing_service_broker="$(cf curl v3/service_brokers | jq --raw-output \
															--arg service_broker_name "${deployment_name}" \
															'.resources[] | select(.name == $service_broker_name) | .name')"
set -e

if [[ -n "${existing_service_broker}" ]]
then
	echo "Service Broker ${existing_service_broker} already exists"
	echo " - cleaning up pr"
	pushd "${script_dir}/../acceptance" > /dev/null
		./cleanup.sh
	popd  > /dev/null
	echo ' - deleting broker'
	cf delete-service-broker -f "${existing_service_broker}"
fi

echo "Creating service broker ${deployment_name} at 'https://${service_broker_name}.${system_domain}'"

autoscaler_service_broker_password=$(credhub get --quiet --name="/bosh-autoscaler/${deployment_name}/service_broker_password")
cf create-service-broker "${deployment_name}" autoscaler-broker-user "$autoscaler_service_broker_password" "https://${service_broker_name}.${system_domain}"

cf logout
