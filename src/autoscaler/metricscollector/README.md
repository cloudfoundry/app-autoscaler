#Metrics Collector

Metrics Collector is one of the components of CF `app-autoscaler`. It is used to collect application metrics from CF loggretator. The current version only supports memory metrics, it will be extended to include other metrics like throughput and response time at a later time.

## Getting started

### System requirements:

* Go 1.11 or above
* Cloud Foundry release 235 or later

### Build and test

1. clone `app-autoscaler` project: `git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git`
1. change directory to `app-autoscaler`
1. run `source .envrc`
1. pull the submodules : `git submodule update --init --recursive`
1. change directory to `src/metricscollector`
1. build the project: `go install ./...
1. run unit test:
  1. download and install [postgreSQL][a]
  1. initialize the database, see [README][b] of the project
  1. install ginko: `go install github.com/onsi/ginkgo/ginkgo`
  1. set environment variable `$DBURL`, e.g. `export DBURL=postgres://postgres:postgres@localhost/autoscaler?sslmode=disable`
  1. run tests: `ginkgo -r -race`
1. regenerate fakes:
  1. install counterfeiter: go install github.com/maxbrunsfeld/counterfeiter
  1. go generate ./...

### Run the metricscollector

Firstly a configuration file needs to be created. Examples can be found under `example-config` directory. Here is an example:

```
cf:
  api: "https://api.bosh-lite.com"
  grant_type: "password"
  username: "admin"
  password: "admin"
server:
  port: 8080
logging:
  level: "info"
db:
  policy_db_url: "postgres://postgres@localhost/autoscaler" 
  metrics_db_url: "postgres://postgres@localhost/autoscaler"
```


The config parameters are explained as below

* cf : cloudfoundry config
 * `api`: API endpoint of cloudfoundry
 * `grant_type`: the grant type when you login into coud foundry, can be "password" or "client_credentials"
 * `username`: the user name when using password grant to login cloudfoundry
 * `password`: the password when using password grant to login cloudfoundry
 * `client_id`: the client id when using client_credentials grant to login cloudfoundry
 * `secret`: the client secret when using client_credentials grant to login cloudfoundry
* server: API sever config
 * `port`: the port API sever will listen to
* logging: config for logging
 * `level`: the level of logging, can be 'debug', 'info', 'error' or 'fatal'
* db : config for database
 * `policy_db_url` : the database url of policy db
 * `metrics_db_url`: the database url of mettics db

To run the metricscollector, use `../../bin/metricscollector -c config_file_name'

## API

Metrics Collector exposes the following APIs for other CF App-Autoscaler components to retrieve metrics.

| PATH                      | METHOD  | Description                              |
|---------------------------|---------|------------------------------------------|
| /v1/apps/{appid}/metrics/memory | GET | Get the latest memroy metric of an application |

[a]: https://www.postgresql.org/download/
[b]: ../../README.md
