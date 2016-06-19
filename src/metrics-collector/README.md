#Metrics-Collector

Metrics-Collector is one of the components of CF `app-autoscaler`. It is used to collect application metrics from CF loggretator. The current version only supports memory metrics, it will be extended to include other metrics like throughput and response time at a later time.

## Getting started

### System requirements:

* Go 1.5 or above
* Cloud Foundry relese 235 or later

### Build and test

1. clone `app-autoscaler` project: `git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git`
1. change directory to `app-autoscaler`
1. pull the submodules : `git submodule update --init --recursive`
1. add `app-autoscaler` directory to your $GOPATH
2. change directory to `src/metrics-collector`
1. build the project: `go build -o out/mc`
1. test the project: 
	1. install ginko: `go install github.com/onsi/ginkgo/ginkgo`
	1. run tests: `ginkgo -r`

### Run the metrics-collector

Firstly a configuration file needs to be created. Examples can be found under `example-config` directory. Here is an example:

```
cf:
  api: "https://api.bosh-lite.com"
  grant_type: "password"
  user: "admin"
  pass: "admin"
server:
  port: 8080
logging:
  level: "info"
```


The config parameters are explained as below

* cf : cloudfoundry config
 * `api`: API endpoint of cloudfoundry
 * `grant_type`: the grant type when you login into coud foundry, can be "password" or "client_credentials"
 * `user`: the user name when using password grant to login cloudfoundry
 * `pass`: the password when using password grant to login cloudfoundry
 * `client_id`: the client id when using client_credentials grant to login cloudfoundry
 * `secret`: the client secret when using client_credentials grant to login cloudfoundry
* server: API sever config
 * `port`: the port API sever will listen to
* logging: config for logging
 * `level`: the level of logging, can be 'debug', 'info', 'error' or 'fatal'

To run the metrics-collector, use `./out/mc -c config_file_name'

## API

Metrics Collector exposes the following APIs for other CF App-Autoscaler components to retrieve metrics.

| PATH                      | METHOD  | Description                              |
|---------------------------|---------|------------------------------------------|
| /v1/apps/{appid}/metrics/memory | GET | Get the latest memroy metric of an application |
