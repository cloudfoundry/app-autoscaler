#!/bin/sh

set -ex

# Install certstrap
go get -v github.com/square/certstrap

# Place keys and certificates here
depot_path="../test-certs"
rm -rf ${depot_path}
mkdir -p ${depot_path}


# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name autoscalerCA --years "20"
mv -f ${depot_path}/autoscalerCA.crt ${depot_path}/autoscaler-ca.crt
mv -f ${depot_path}/autoscalerCA.key ${depot_path}/autoscaler-ca.key

# metricscollector certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metricscollector --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricscollector --CA autoscaler-ca --years "20"

# scalingengine certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name scalingengine --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scalingengine --CA autoscaler-ca --years "20"

# eventgenerator certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name eventgenerator --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign eventgenerator --CA autoscaler-ca --years "20"

# servicebroker certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name servicebroker --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker --CA autoscaler-ca --years "20"
# servicebroker certificate for internal
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name servicebroker_internal --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker_internal --CA autoscaler-ca --years "20"

# api certificate for internal connection
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name api --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api --CA autoscaler-ca --years "20"

# api certificate for public api
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name api_public --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api_public --CA autoscaler-ca --years "20"

# scheduler certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name scheduler --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scheduler --CA autoscaler-ca --years "20"
openssl pkcs12 -export -in ${depot_path}/scheduler.crt -inkey ${depot_path}/scheduler.key -out ${depot_path}/scheduler.p12 -name scheduler -password pass:123456
keytool -importcert -alias autoscaler -file ${depot_path}/autoscaler-ca.crt -keystore ${depot_path}/autoscaler.truststore -storeType pkcs12 -storepass 123456 -noprompt
