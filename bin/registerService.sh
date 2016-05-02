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
scheme=${apiServerURI%:*}
brokerURI=$scheme://AutoScalingServiceBroker.$appDomain

cf marketplace -s $serviceName
if [ $? -ne 0 ]; then
	echo "cf create-service-broker $serviceName <brokerUserName> <brokerPassword> $brokerURI"
	cf create-service-broker $serviceName $brokerUsername $brokerPassword $brokerURI
	cf enable-service-access $serviceName
else
	echo "cf update-service-broker $serviceName <brokerUserName> <brokerPassword> $brokerURI"
	cf update-service-broker $serviceName $brokerUsername $brokerPassword $brokerURI
fi
