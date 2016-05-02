#!/bin/bash
if [[ "$#" != 1 ]]; then
  echo 'An environment is required'
	echo "for example, $0 myenv"
  exit 1
fi

basedir=$(cd "$(dirname "$0")"; pwd)
appAutoScaler="${basedir}/.."
envProperties="${appAutoScaler}/profiles/$1.properties"
if [ ! -f  $envProperties ]; then
	echo "The file '$envProperties' does not exist"
	exit 1
fi

source $envProperties

appDomain=${apiServerURI#*://*.}

cf push AutoScaling -p $appAutoScaler/server/target/server-*.war -d $appDomain
cf push AutoScalingAPI -p $appAutoScaler/api/target/api-*.war -d $appDomain
cf push AutoScalingServiceBroker -p $appAutoScaler/servicebroker/target/servicebroker-*.war -d $appDomain
