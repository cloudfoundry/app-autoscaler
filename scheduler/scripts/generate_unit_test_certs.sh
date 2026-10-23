#!/bin/bash
set -e
script_dir="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

# Place keys and certificates here
depot_path="${script_dir}/../src/test/resources/certs"

rm -rf "${depot_path}"
mkdir -p "${depot_path}"

CERTSTRAP="go run github.com/square/certstrap@v1.2.0"

# CA to distribute to autoscaler certs
${CERTSTRAP} --depot-path "${depot_path}" init --passphrase '' --common-name testCA
mv -f "${depot_path}"/testCA.crt "${depot_path}"/test-ca.crt
mv -f "${depot_path}"/testCA.key "${depot_path}"/test-ca.key


# scalingengine certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --common-name test-scalingengine --domain localhost
${CERTSTRAP} --depot-path "${depot_path}" sign test-scalingengine --CA test-ca

# scheduler certificate
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' --common-name test-scheduler --domain localhost
${CERTSTRAP} --depot-path "${depot_path}" sign test-scheduler --CA test-ca

# XFCC client certificate for HttpAuthFilter tests.
# The filter authenticates by substring-matching the Subject OU against
# "space:<guid>" and "organization:<guid>"; the CF app-identity cert carries
# both in a single OU value literally formatted as
# "OU=space:<space>+OU=organization:<org>" (see testhelpers/certs.go). The
# space/org GUIDs below match cfserver.validSpaceGuid/validOrgGuid used by the
# filter tests.
${CERTSTRAP} --depot-path "${depot_path}" request-cert --passphrase '' \
	--common-name localhost \
	--organization 'My Organization' \
	--organizational-unit 'OU=space:validspace+OU=organization:validorg' \
	--domain localhost
${CERTSTRAP} --depot-path "${depot_path}" sign localhost --CA test-ca
mv -f "${depot_path}"/localhost.crt "${depot_path}"/xfcc-valid.crt
mv -f "${depot_path}"/localhost.key "${depot_path}"/xfcc-valid.key

keytool -importcert -alias autoscaler -file "${depot_path}"/test-ca.crt -keystore "${depot_path}"/test.truststore -storeType pkcs12 -storepass 123456 -noprompt

openssl pkcs12 -export -in "${depot_path}"/test-scheduler.crt -inkey "${depot_path}"/test-scheduler.key -out "${depot_path}"/test-scheduler.p12 -name test-scheduler -password pass:123456
openssl pkcs12 -export -in "${depot_path}"/test-scalingengine.crt -inkey "${depot_path}"/test-scalingengine.key -out "${depot_path}"/fake-scalingengine.p12 -name fake-scalingengine -password pass:123456

# EC and PKCS8 private keys for Spring Boot PEM compatibility tests.
openssl ecparam -name prime256v1 -genkey -noout -out "${depot_path}"/test-ec.key
openssl pkcs8 -topk8 -nocrypt -in "${depot_path}"/test-scheduler.key -out "${depot_path}"/test-rsa-pkcs8.key
