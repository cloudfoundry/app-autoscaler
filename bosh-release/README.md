# Auto-Scaler BOSH Release

This is a release repository for Auto-Scaler that deploys the Auto-Scaler service which includes four components: Service Broker, API Server, Scaling Server and CouchDB


## Release Contents

### Jobs

The release has the following jobs:

* `as_broker`: implements the service borker interface of Cloud Foundry, running as a web application
* `as_api`:  provides APIs for user to interact with the Auto-Scaling service, running as a web application
* `as_server`: provides the core functions of metrics collection, evaluation and action taken, running as a web application
* `couchdb`: the database tier for data persistency

packages:

* jre : `as_broker`,  `as_server`, `as_api`
* tomcat:  `as_broker`,  `as_api`, `as_server`
* broker_war: `as_broker`
* api_war:  `as_api`
* server_war: `as_server`
* couchdb: `couchdb`

## Deploy 

coming
