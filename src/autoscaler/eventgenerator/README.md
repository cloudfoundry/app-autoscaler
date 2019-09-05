#Data Aggregator

Data Aggregator is one of the components of CF `app-autoscaler`. 

## Getting started

### System requirements:

* Go 1.11 or above

### Build and test

1. clone `app-autoscaler` project: `git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git`
1. change directory to `app-autoscaler`
1. run `source .envrc`
1. pull the submodules : `git submodule update --init --recursive`
1. change directory to eventgenerator root path : `autoscaler/src/autoscaler/eventgenerator`
1. build the project: `go install ./...
1. run unit test:
  1. download and install [postgreSQL][a]
  1. initialize the database, see [README][b] of the project
  1. install ginko: `go install github.com/onsi/ginkgo/ginkgo`
  1. set environment variable `$DBURL`, e.g. `export DBURL=postgres://postgres:postgres@localhost/autoscaler?sslmode=disable`
  1. run tests: `ginkgo -r -race`


[a]: https://www.postgresql.org/download/
[b]: ../../README.md
