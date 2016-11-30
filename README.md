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
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=api/db/api.db.changelog.yml update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=servicebroker/db/servicebroker.db.changelog.json update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=scheduler/db/scheduler.changelog-master.yaml update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=scheduler/db/quartz.changelog-master.yaml update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/metricscollector/db/metricscollector.db.changelog.yml update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/eventgenerator/db/dataaggregator.db.changelog.yml update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/scalingengine/db/scalingengine.db.changelog.yml update
```
### Generate TLS Certificates
```shell
./scripts/generate_test_certs.sh
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
pushd src/autoscaler
ginkgo -r -race -randomizeAllSpecs
popd

pushd scheduler
mvn test
popd
```

## Integration tests
```shell
pushd api
npm install
popd

pushd scheduler
mvn package -DskipTests
popd

go install github.com/onsi/ginkgo/ginkgo
export DBURL=postgres://postgres@localhost/autoscaler?sslmode=disable
ginkgo -r -race -randomizeAllSpecs src/integration
```
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
