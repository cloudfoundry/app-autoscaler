#!/bin/sh

set -ex

# Install certstrap
go get -v github.com/square/certstrap

# Place keys and certificates here
depot_path="../src/test/resources/certs"
rm -rf ${depot_path}
mkdir -p ${depot_path}

# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name testCA
mv -f ${depot_path}/testCA.crt ${depot_path}/test-ca.crt
mv -f ${depot_path}/testCA.key ${depot_path}/test-ca.key


# scalingengine certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name test-scalingengine --domain localhost
certstrap --depot-path ${depot_path} sign test-scalingengine --CA test-ca

# scheduler certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name test-scheduler --domain localhost
certstrap --depot-path ${depot_path} sign test-scheduler --CA test-ca

keytool -importcert -alias autoscaler -file ${depot_path}/test-ca.crt -keystore ${depot_path}/test.truststore -storeType pkcs12 -storepass 123456 -noprompt

openssl pkcs12 -export -in ${depot_path}/test-scheduler.crt -inkey ${depot_path}/test-scheduler.key -out ${depot_path}/test-scheduler.p12 -name test-scheduler -password pass:123456
openssl pkcs12 -export -in ${depot_path}/test-scalingengine.crt -inkey ${depot_path}/test-scalingengine.key -out ${depot_path}/fake-scalingengine.p12 -name fake-scalingengine -password pass:123456

