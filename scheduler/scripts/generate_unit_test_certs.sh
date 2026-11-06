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

# Dedicated XFCC intermediate. Keep the existing server TLS fixtures unchanged.
openssl req -new -newkey rsa:2048 -nodes -sha256 \
	-subj '/CN=instanceIdentityCA' \
	-keyout "${depot_path}/xfcc-issuer.key" -out "${depot_path}/xfcc-issuer.csr"
cat > "${depot_path}/xfcc-issuer.ext" <<'EOF'
basicConstraints=critical,CA:TRUE,pathlen:0
keyUsage=critical,keyCertSign,cRLSign
EOF
openssl x509 -req -sha256 -days 365 \
	-in "${depot_path}/xfcc-issuer.csr" \
	-CA "${depot_path}/test-ca.crt" -CAkey "${depot_path}/test-ca.key" -CAcreateserial \
	-extfile "${depot_path}/xfcc-issuer.ext" -out "${depot_path}/xfcc-issuer.crt"

cp "${depot_path}/xfcc-issuer.ext" "${depot_path}/xfcc-server-issuer.ext"
printf '%s\n' 'extendedKeyUsage=serverAuth' >> "${depot_path}/xfcc-server-issuer.ext"
openssl x509 -req -sha256 -days 365 \
	-in "${depot_path}/xfcc-issuer.csr" \
	-CA "${depot_path}/test-ca.crt" -CAkey "${depot_path}/test-ca.key" -CAcreateserial \
	-extfile "${depot_path}/xfcc-server-issuer.ext" -out "${depot_path}/xfcc-server-issuer.crt"

openssl req -new -newkey rsa:2048 -nodes -sha256 \
	-subj '/CN=localhost/O=My Organization/OU=OU=space:validspace+OU=organization:validorg' \
	-keyout "${depot_path}/xfcc-valid.key" -out "${depot_path}/xfcc-valid.csr"

sign_xfcc_leaf() {
	local name="$1" extensions="$2"
	printf '%s\n' 'basicConstraints=critical,CA:FALSE' 'keyUsage=critical,digitalSignature' \
		"${extensions}" > "${depot_path}/${name}.ext"
	openssl x509 -req -sha256 -days 365 \
		-in "${depot_path}/xfcc-valid.csr" \
		-CA "${depot_path}/xfcc-issuer.crt" -CAkey "${depot_path}/xfcc-issuer.key" -CAcreateserial \
		-extfile "${depot_path}/${name}.ext" -out "${depot_path}/${name}.crt"
}

sign_xfcc_leaf xfcc-valid 'extendedKeyUsage=clientAuth'
sign_xfcc_leaf xfcc-no-eku ''
sign_xfcc_leaf xfcc-any-eku 'extendedKeyUsage=anyExtendedKeyUsage'
sign_xfcc_leaf xfcc-server-eku 'extendedKeyUsage=serverAuth'
sign_xfcc_leaf xfcc-critical '1.2.3.4=critical,DER:05:00'

mkdir -p "${depot_path}/cf-system-certificates" "${depot_path}/cf-instance-credentials"
cp "${depot_path}/test-ca.crt" "${depot_path}/cf-system-certificates/app-root-ca.crt"
cat "${depot_path}/xfcc-valid.crt" "${depot_path}/xfcc-issuer.crt" \
	> "${depot_path}/cf-instance-credentials/instance.crt"

keytool -importcert -alias autoscaler -file "${depot_path}"/test-ca.crt -keystore "${depot_path}"/test.truststore -storeType pkcs12 -storepass 123456 -noprompt

openssl pkcs12 -export -in "${depot_path}"/test-scheduler.crt -inkey "${depot_path}"/test-scheduler.key -out "${depot_path}"/test-scheduler.p12 -name test-scheduler -password pass:123456
openssl pkcs12 -export -in "${depot_path}"/test-scalingengine.crt -inkey "${depot_path}"/test-scalingengine.key -out "${depot_path}"/fake-scalingengine.p12 -name fake-scalingengine -password pass:123456

# EC and PKCS8 private keys for Spring Boot PEM compatibility tests.
openssl ecparam -name prime256v1 -genkey -noout -out "${depot_path}"/test-ec.key
openssl pkcs8 -topk8 -nocrypt -in "${depot_path}"/test-scheduler.key -out "${depot_path}"/test-rsa-pkcs8.key
