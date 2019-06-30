#!/bin/sh

set -ex

# Install certstrap
go get -v github.com/square/certstrap

# Place keys and certificates here
depot_path="../test-certs"
rm -rf ${depot_path}
mkdir -p ${depot_path}


# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name autoscalerCA --expires "20 years"
mv -f ${depot_path}/autoscalerCA.crt ${depot_path}/autoscaler-ca.crt
mv -f ${depot_path}/autoscalerCA.key ${depot_path}/autoscaler-ca.key

# CA to distribute to dummy loggregator_agent certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name loggregatorCA --expires "20 years"
mv -f ${depot_path}/loggregatorCA.crt ${depot_path}/loggregator-ca.crt
mv -f ${depot_path}/loggregatorCA.key ${depot_path}/loggregator-ca.key

# metricscollector certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metricscollector --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricscollector --CA autoscaler-ca --expires "20 years"

# scalingengine certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name scalingengine --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scalingengine --CA autoscaler-ca --expires "20 years"

# eventgenerator certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name eventgenerator --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign eventgenerator --CA autoscaler-ca --expires "20 years"

# servicebroker certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name servicebroker --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker --CA autoscaler-ca --expires "20 years"
# servicebroker certificate for internal
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name servicebroker_internal --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker_internal --CA autoscaler-ca --expires "20 years"

# api certificate for internal connection
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name api --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api --CA autoscaler-ca --expires "20 years"

# api certificate for public api
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name api_public --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api_public --CA autoscaler-ca --expires "20 years"

# scheduler certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name scheduler --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scheduler --CA autoscaler-ca --expires "20 years"
openssl pkcs12 -export -in ${depot_path}/scheduler.crt -inkey ${depot_path}/scheduler.key -out ${depot_path}/scheduler.p12 -name scheduler -password pass:123456
keytool -importcert -alias autoscaler -file ${depot_path}/autoscaler-ca.crt -keystore ${depot_path}/autoscaler.truststore -storeType pkcs12 -storepass 123456 -noprompt


# # loggregator test server certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name reverselogproxy
certstrap --depot-path ${depot_path} sign reverselogproxy --CA autoscaler-ca --expires "20 years"

# loggregator test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name reverselogproxy_client
certstrap --depot-path ${depot_path} sign reverselogproxy_client --CA autoscaler-ca --expires "20 years"

# metricserver test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metricserver --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricserver --CA autoscaler-ca --expires "20 years"

# metricserver test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metricserver_client
certstrap --depot-path ${depot_path} sign metricserver_client --CA autoscaler-ca --expires "20 years"

# metricsforwarder certificate for loggregator_agent
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metron
certstrap --depot-path ${depot_path} sign metron --CA loggregator-ca --expires "20 years"
