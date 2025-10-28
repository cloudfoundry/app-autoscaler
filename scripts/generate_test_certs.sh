#!/bin/bash

set -e

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Place keys and certificates here
depot_path="${script_dir}/../test-certs"
rm -rf "${depot_path}"
mkdir -p "${depot_path}"

CERTSTRAP="go run github.com/square/certstrap@v1.3.0"

# CA to distribute to autoscaler certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name autoscalerCA --years "20"
mv -f "${depot_path}"/autoscalerCA.crt "${depot_path}"/autoscaler-ca.crt
mv -f "${depot_path}"/autoscalerCA.key "${depot_path}"/autoscaler-ca.key

# CA to distribute to dummy loggregator_agent certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name loggregatorCA --years "20"
mv -f "${depot_path}"/loggregatorCA.crt "${depot_path}"/loggregator-ca.crt
mv -f "${depot_path}"/loggregatorCA.key "${depot_path}"/loggregator-ca.key

# CA to distribute to dummy gorouter ca certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name gorouterCA --years "20"
mv -f "${depot_path}"/gorouterCA.crt "${depot_path}"/gorouter-ca.crt
mv -f "${depot_path}"/gorouterCA.key "${depot_path}"/gorouter-ca.key

# CA to distribute to dummy syslog emitter certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name LogCacheSyslogServerCA --years "20"
mv -f "${depot_path}"/LogCacheSyslogServerCA.crt "${depot_path}"/log-cache-syslog-server-ca.crt
mv -f "${depot_path}"/LogCacheSyslogServerCA.key "${depot_path}"/log-cache-syslog-server-ca.key

# CA for local testing mTLS certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name validMTLSLocalCA --years "20"
mv -f "${depot_path}"/validMTLSLocalCA.crt "${depot_path}"/valid-mtls-local-ca-1.crt
mv -f "${depot_path}"/validMTLSLocalCA.key "${depot_path}"/valid-mtls-local-ca-1.key
rm -f "${depot_path}"/validMTLSLocalCA.crl

${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name validMTLSLocalCA --years "20"
mv -f "${depot_path}"/validMTLSLocalCA.crt "${depot_path}"/valid-mtls-local-ca-2.crt
mv -f "${depot_path}"/validMTLSLocalCA.key "${depot_path}"/valid-mtls-local-ca-2.key

# CA for local testing mTLS certs (another CA for validating verification)
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name invalidMTLSLocalCA --years "20"
mv -f "${depot_path}"/invalidMTLSLocalCA.crt "${depot_path}"/invalid-mtls-local-ca.crt
mv -f "${depot_path}"/invalidMTLSLocalCA.key "${depot_path}"/invalid-mtls-local-ca.key

# metricscollector certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain metricscollector --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign metricscollector --CA autoscaler-ca --years "20"

# scalingengine certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain scalingengine --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign scalingengine --CA autoscaler-ca --years "20"

# eventgenerator certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain eventgenerator --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign eventgenerator --CA autoscaler-ca --years "20"

# servicebroker certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain servicebroker --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign servicebroker --CA autoscaler-ca --years "20"
# servicebroker certificate for internal
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain servicebroker_internal --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign servicebroker_internal --CA autoscaler-ca --years "20"

# api certificate for internal connection
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain api --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign api --CA autoscaler-ca --years "20"

# api certificate for public api
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain api_public --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign api_public --CA autoscaler-ca --years "20"

# scheduler certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain scheduler --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign scheduler --CA autoscaler-ca --years "20"

# # loggregator test server certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain reverselogproxy
${CERTSTRAP} --depot-path "${depot_path}" sign reverselogproxy --CA autoscaler-ca --years "20"

# loggregator test client certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain reverselogproxy_client
${CERTSTRAP} --depot-path "${depot_path}" sign reverselogproxy_client --CA autoscaler-ca --years "20"

# metricsforwarder certificate for loggregator_agent
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain metron
${CERTSTRAP} --depot-path "${depot_path}" sign metron --CA loggregator-ca --years "20"

# metricsforwarder certificate for log-cache-syslog-server
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain cf-app
${CERTSTRAP} --depot-path "${depot_path}" sign cf-app --CA log-cache-syslog-server-ca --years "20"

# log-cache certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain log-cache --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign log-cache --CA autoscaler-ca --years "20"

# database certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain postgres,mysql --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign postgres --CA autoscaler-ca --years "20"

# gorouter client certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain gorouter --ip 127.0.0.1
${CERTSTRAP} --depot-path "${depot_path}" sign gorouter --CA gorouter-ca --years "20"

# mTLS client certificate for local testing
## certstrap with multiple OU not working at the moment. Pull request is created in the upstream. Therefore, using openssl at the moment
## https://github.com/square/certstrap/pull/120
#${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --domain local_client --ou "app-id:an-app-id,organization:ORG-GUID,space:SPACE-GUID"

OPENSSL_VERSION=$(openssl version)
if [[ "$OPENSSL_VERSION" == LibreSSL* ]]; then
	echo "OpenSSL needs to be used rather than LibreSSL"
	exit 1
fi
# valid certificate
echo "${depot_path}"
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out "${depot_path}"/validmtls_client-1.csr
openssl x509 -req -in "${depot_path}"/validmtls_client-1.csr -CA "${depot_path}"/valid-mtls-local-ca-1.crt -CAkey "${depot_path}"/valid-mtls-local-ca-1.key -CAcreateserial -out "${depot_path}"/validmtls_client-1.crt -days 365 -sha256
openssl  req -new -newkey rsa:2048  -nodes -subj "/CN=sap.com/O=SAP SE/OU=organization:AB1234ORG/OU=app:an-app-id/OU=space:AB1234SPACE" -out "${depot_path}"/validmtls_client-2.csr
openssl x509 -req -in "${depot_path}"/validmtls_client-2.csr -CA "${depot_path}"/valid-mtls-local-ca-2.crt -CAkey "${depot_path}"/valid-mtls-local-ca-2.key -CAcreateserial -out "${depot_path}"/validmtls_client-2.crt -days 365 -sha256

# remove the generated key
rm privkey.pem
