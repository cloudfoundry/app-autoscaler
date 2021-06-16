#!/bin/sh

set -ex

# Install certstrap
go install github.com/square/certstrap@v1.2.0

# Place keys and certificates here
depot_path="../test-certs"
rm -rf ${depot_path}
mkdir -p ${depot_path}


# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name autoscalerCA --years "20"
mv -f ${depot_path}/autoscalerCA.crt ${depot_path}/autoscaler-ca.crt
mv -f ${depot_path}/autoscalerCA.key ${depot_path}/autoscaler-ca.key

# CA to distribute to dummy loggregator_agent certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name loggregatorCA --years "20"
mv -f ${depot_path}/loggregatorCA.crt ${depot_path}/loggregator-ca.crt
mv -f ${depot_path}/loggregatorCA.key ${depot_path}/loggregator-ca.key

# metricscollector certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain metricscollector --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricscollector --CA autoscaler-ca --years "20"

# scalingengine certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain scalingengine --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scalingengine --CA autoscaler-ca --years "20"

# eventgenerator certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain eventgenerator --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign eventgenerator --CA autoscaler-ca --years "20"

# servicebroker certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain servicebroker --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker --CA autoscaler-ca --years "20"
# servicebroker certificate for internal
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain servicebroker_internal --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker_internal --CA autoscaler-ca --years "20"

# api certificate for internal connection
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain api --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api --CA autoscaler-ca --years "20"

# api certificate for public api
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain api_public --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign api_public --CA autoscaler-ca --years "20"

# scheduler certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain scheduler --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scheduler --CA autoscaler-ca --years "20"
openssl pkcs12 -export -in ${depot_path}/scheduler.crt -inkey ${depot_path}/scheduler.key -out ${depot_path}/scheduler.p12 -name scheduler -password pass:123456
keytool -importcert -alias autoscaler -file ${depot_path}/autoscaler-ca.crt -keystore ${depot_path}/autoscaler.truststore -storeType pkcs12 -storepass 123456 -noprompt


# # loggregator test server certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain reverselogproxy
certstrap --depot-path ${depot_path} sign reverselogproxy --CA autoscaler-ca --years "20"

# loggregator test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain reverselogproxy_client
certstrap --depot-path ${depot_path} sign reverselogproxy_client --CA autoscaler-ca --years "20"

# metricserver test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain metricserver --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricserver --CA autoscaler-ca --years "20"

# metricserver test client certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain metricserver_client
certstrap --depot-path ${depot_path} sign metricserver_client --CA autoscaler-ca --years "20"

# metricsforwarder certificate for loggregator_agent
certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain metron
certstrap --depot-path ${depot_path} sign metron --CA loggregator-ca --years "20"
