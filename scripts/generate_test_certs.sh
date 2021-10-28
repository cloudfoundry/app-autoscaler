#!/bin/bash

set -e
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Place keys and certificates here
depot_path="${script_dir}/../test-certs"
rm -rf ${depot_path}
mkdir -p ${depot_path}

OS=$(uname || "Win")
if [ ${OS} = "Darwin" ]; then
  which certstrap >/dev/null || brew install certstrap
else
  # Install certstrap
  which certstrap >/dev/null || go get -v github.com/square/certstrap
fi

# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name autoscalerCA --years "20"
mv -f ${depot_path}/autoscalerCA.crt ${depot_path}/autoscaler-ca.crt
mv -f ${depot_path}/autoscalerCA.key ${depot_path}/autoscaler-ca.key

# CA to distribute to dummy loggregator_agent certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name loggregatorCA --years "20"
mv -f ${depot_path}/loggregatorCA.crt ${depot_path}/loggregator-ca.crt
mv -f ${depot_path}/loggregatorCA.key ${depot_path}/loggregator-ca.key

# CA for local testing mTLS certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name validMTLSLocalCA --years "20"
mv -f ${depot_path}/validMTLSLocalCA.crt ${depot_path}/valid-mtls-local-ca-1.crt
mv -f ${depot_path}/validMTLSLocalCA.key ${depot_path}/valid-mtls-local-ca-1.key
rm -f ${depot_path}/validMTLSLocalCA.crl
certstrap --depot-path ${depot_path} init --passphrase '' --common-name validMTLSLocalCA --years "20"
mv -f ${depot_path}/validMTLSLocalCA.crt ${depot_path}/valid-mtls-local-ca-2.crt
mv -f ${depot_path}/validMTLSLocalCA.key ${depot_path}/valid-mtls-local-ca-2.key

#CA with multiple certs. The \n\n is to enable testing of extra white space in the multiple pem cert files
{ printf "\n\n"; \
  cat "${depot_path}/valid-mtls-local-ca-1.crt";\
  printf "\t\n\n" ; \
  cat "${depot_path}/valid-mtls-local-ca-2.crt";\
  printf "\n  \n"; } > "${depot_path}/valid-mtls-local-ca-combined.crt"

# CA for local testing mTLS certs (another CA for validating verification)
certstrap --depot-path ${depot_path} init --passphrase '' --common-name invalidMTLSLocalCA --years "20"
mv -f ${depot_path}/invalidMTLSLocalCA.crt ${depot_path}/invalid-mtls-local-ca.crt
mv -f ${depot_path}/invalidMTLSLocalCA.key ${depot_path}/invalid-mtls-local-ca.key

# empty CA file  for local testing mTLS certs
{ printf "   \n\n"; \
  printf "\t\n\n" ; \
  printf "\t\n\n" ; \
  printf "\t\n\n" ; \
  printf "\n  \n"; } > "${depot_path}/empty-mtls-local-ca.crt"

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

# mTLS client certificate for local testing
## certstrap with multiple OU not working at the moment. Pull request is created in the upstream. Therefore, using openssl at the moment
## https://github.com/square/certstrap/pull/120
#certstrap --depot-path ${depot_path} request-cert --passphrase '' --domain local_client --ou "app-id:an-app-id,organization:ORG-GUID,space:SPACE-GUID"

OPENSSL_VERSION=$(openssl version)
if [[ "$OPENSSL_VERSION" == LibreSSL* ]]; then
	echo "OpenSSL needs to be used rather than LibreSSL"
	exit 1
fi
# valid certificate
echo ${depot_path}
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out ${depot_path}/validmtls_client-1.csr
openssl x509 -req -in "${depot_path}"/validmtls_client-1.csr -CA "${depot_path}"/valid-mtls-local-ca-1.crt -CAkey "${depot_path}"/valid-mtls-local-ca-1.key -CAcreateserial -out "${depot_path}"/validmtls_client-1.crt -days 365 -sha256
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out ${depot_path}/validmtls_client-2.csr
openssl x509 -req -in "${depot_path}"/validmtls_client-2.csr -CA "${depot_path}"/valid-mtls-local-ca-2.crt -CAkey "${depot_path}"/valid-mtls-local-ca-2.key -CAcreateserial -out "${depot_path}"/validmtls_client-2.crt -days 365 -sha256

## invalid certificate ( with invalid CA)
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out "${depot_path}"/invalidmtls_client.csr
openssl x509 -req -in "${depot_path}"/invalidmtls_client.csr -CA "${depot_path}"/invalid-mtls-local-ca.crt -CAkey "${depot_path}"/invalid-mtls-local-ca.key -CAcreateserial -out "${depot_path}"/invalidmtls_client.crt -days 365 -sha256

## unsigned certificate
openssl  req -x509 -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out "${depot_path}"/nosignmtls_client.crt
#
## expired certificate
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out "${depot_path}"/expiredmtls_client.csr
openssl x509 -req -in "${depot_path}"/expiredmtls_client.csr -CA "${depot_path}"/valid-mtls-local-ca-1.crt -CAkey "${depot_path}"/valid-mtls-local-ca-1.key -CAcreateserial -out "${depot_path}"/expiredmtls_client.crt -days 0 -sha256

# remove the generated key
rm privkey.pem
