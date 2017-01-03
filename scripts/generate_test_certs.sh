#!/bin/sh

set -ex

# Install certstrap
go get -v github.com/square/certstrap

# Place keys and certificates here
depot_path="testcerts"
rm -rf ${depot_path}
mkdir -p ${depot_path}


# CA to distribute to autoscaler certs
certstrap --depot-path ${depot_path} init --passphrase '' --common-name autoscalerCA
mv -f ${depot_path}/autoscalerCA.crt ${depot_path}/autoscaler-ca.crt
mv -f ${depot_path}/autoscalerCA.key ${depot_path}/autoscaler-ca.key

# metricscollector certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name metricscollector --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign metricscollector --CA autoscaler-ca

# scalingengine certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name scalingengine --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign scalingengine --CA autoscaler-ca

# eventgenerator certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name eventgenerator --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign eventgenerator --CA autoscaler-ca

# servicebroker certificate
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name servicebroker --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign servicebroker --CA autoscaler-ca