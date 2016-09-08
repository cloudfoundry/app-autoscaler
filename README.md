<link href="https://raw.github.com/clownfart/Markdown-CSS/master/markdown.css" rel="stylesheet"></link>
[![Build Status](https://runtime-og.ci.cf-app.com/api/v1/pipelines/autoscaler/jobs/unit-tests/badge?ts=1)](https://runtime-og.ci.cf-app.com/pipelines/autoscaler)

# CF-AutoScaler

This is an incubation project for Cloud Foundry. You can follow the development progress on [Pivotal Tracker][t].

The `CF-AutoScaler` provides the capability to adjust the computation resources for Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The `CF-AutoScaler` is provided as a Cloud Foundry service offering. Any application bound with `CF-AutoScaler` service will be able to use it.

## Getting Started

System requirements:

* Java 8 or above
* [Apache Maven][b] 3
* Node 6.2 or above
* NPM 3.9.5 or above
* [Cloud Foundry cf command line] [f]
* [Cloud Foundry UAA command line client][u]

Database requirement:

The `CF-AutoScaler` uses Postgres as the backend data store.

To get started, clone this project:

```shell
$ git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git
$ cd app-autoscaler
```

The AutoScaler has multiple components.

* `api` : provides public APIs to manage scaling policy
* `servicebroker`: implements the [Cloud Foundry service broker API][k]
* `metricscollector`: collect container's memory usage



`CF-AutoScaler` invokes Cloud controller API to trigger scaling on target application. To achieve this, a UAA client id with  authorities `cloud_controller.read,cloud_controller.admin` is needed for the Cloud Foundry environment `CF-AutoScaler` is registered with. You can create it using UAA command line client, make sure the client ID and secret are the ones you configured when you package the .war files

```shell
uaac target http://uaa.<cf-domain>
uaac token client get admin -s <cf uaa admin secret>
uaac client add cf-autoscaler-client \
	--name cf-autoscaler \
    --authorized_grant_types client_credentials \
    --authorities cloud_controller.read,cloud_controller.admin \
    --secret cf-autoscaler-client-secret
```

The following sections describe how to test, deploy and run `CF-AutoScaler` service manually.

### Initialize the Database
```shell
createuser postgres -s
psql postgres://postgres@127.0.0.1:5432 -c 'DROP DATABASE IF EXISTS autoscaler'
psql postgres://postgres@127.0.0.1:5432 -c 'CREATE DATABASE autoscaler'

mvn package
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=api/db/api.db.changelog.yml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=servicebroker/db/servicebroker.db.changelog.json --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=src/metricscollector/db/metricscollector.db.changelog.yml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=src/eventgenerator/db/dataaggregator.db.changelog.yml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=scheduler/db/scheduler.changelog-master.yaml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=scheduler/db/quartz.changelog-master.yaml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
```

### Run Unit Tests

For all these three `CF-AutoScaler` components, a unit test `unittest.properties` under 'profiles' directory has been created to define the settings for unit test.

## Unit tests
```shell
pushd api
npm install
npm test
popd

pushd servicebroker
npm install
npm test
popd

go install github.com/onsi/ginkgo/ginkgo
export DBURL=postgres://postgres@localhost/autoscaler?sslmode=disable
pushd src/metricscollector
ginkgo -r -race -randomizeAllSpecs
popd
pushd src/cf
ginkgo -r -race -randomizeAllSpecs
popd
pushd src/db
ginkgo -r -race -randomizeAllSpecs
popd
pushd src/eventgenerator
ginkgo -r -race -randomizeAllSpecs
popd

pushd scheduler
mvn test
popd
```

### Configure and Package

All the `CF-AutoScaler` components are configured through a single properties file. To create your own settings, copy the following properties and change the appropriate values for your environment.

```
# CloudFoundry settings
cfUrl=api.my.domain
cfClientId=cf-autoscaler-client
cfClientSecret=cf-autoscaler-client-secret

# Service broker settings
serviceName=CF-AutoScaler
brokerUsername=admin
brokerPassword=admin

# http basic authentication settings between the CF-Autoscaler components
internalAuthUsername=admin
internalAuthPassword=admin

# scaling and api server URI settings
scalingServerURIList=https://autoscaling.my.app.domain
apiServerURI=https://autoscalingapi.my.app.domain

# couchdb settings
couchdbUsername=autoscaler
couchdbPassword=openopen
couchdbHost=xxx.xxx.xxx.xxx
couchdbPort=5984
couchdbServerDBName=couchdb-scaling
couchdbMetricDBPrefix=couchdb-scalingmetric
couchdbBrokerDBName=couchdb-scalingbroker

# metrics settings
reportInterval=120
```

Assume you want to name your environment, "myenv". Create a properties file in `app-autoscaler/profiles/myenv.properties` with all the properties defined above.

To package all the .war files for deployment, use mvn package. The *.war file can be found in each folder `{project}/target` directory.

```shell
mvn clean package -Denv=myenv -DskipTests
```

### Deploy `CF-AutoScaler` service

Push the .war package of each `CF-AutoScaler` components to Cloud Foundry to deploy `CF-AutoScaler` service.

The deployment assumes that there is a couchdb instance available with the configured host, port, username and password, and the couchdb can be accessed from an application running within Cloud Foundry.

```shell
bin/deploy.sh myenv
```

### Register `CF-AutoScaler` service broker

Register `CF-AutoScaler` with Cloud Foundry.

```shell
bin/registerService.sh myenv
```

## Test your 'CF-AutoScaler' deployment
See [src/acceptance/README.md](src/acceptance/README.md)

## Use `CF-AutoScaler`

Now, you can play with `CF-AutoScaler`.
Create a `CF-AutoScaler` service, and bind to you application

```shell
cf create-service CF-AutoScaler free <service_instance>
cf bind-service <app> <service_instance>
```

Then refer to [API_usage.rst][a] to manage the scaling policy of your application, retrieve metrics and scaling histories.

## Tips for development
1. To run acceptance tests faster, reduce the `reportInterval` in your config. You will also need to adjust the same value in your integration config.
1. Logs are available locally with each application instance. For example, to retreive the logs for the API server, run `cf files AutoScalingAPI app/autoscaling_api.log`. They will roll so there may be additional files to examine.
1. Depending on where you deploy CouchDB, the AutoScaler service container's
   may not have network connectivity to it. To allow access, create and bind
   a security group.

```shell
cat > my-security-group.json <<EOF
[
  {
    "protocol": "tcp",
    "destination": "0.0.0.0/0",
    "ports": "5984"
  }
]
EOF

cf create-security-group couchdb my-security-group.json
cf bind-running-security-group couchdb
```
Restart the applications if needed

## License

This project is released under version 2.0 of the [Apache License][l].


[a]: docs/API_usage.rst
[b]: https://maven.apache.org/
[c]: http://couchdb.apache.org/
[d]: http://www.eclipse.org/m2e/
[e]: http://www.cloudant.com
[f]: https://github.com/cloudfoundry/cli/releases
[k]: http://docs.cloudfoundry.org/services/api.html
[l]: LICENSE
[t]: https://www.pivotaltracker.com/projects/1566795
[u]: https://github.com/cloudfoundry/cf-uaac
