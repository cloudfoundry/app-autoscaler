# App-AutoScaler [![Build Status](https://github.com/cloudfoundry/app-autoscaler/actions/workflows/postgres.yaml/badge.svg)](https://github.com/cloudfoundry/app-autoscaler/actions/workflows/postgres.yaml) [![Build Status](https://github.com/cloudfoundry/app-autoscaler/actions/workflows/mysql.yaml/badge.svg)](https://github.com/cloudfoundry/app-autoscaler/actions/workflows/mysql.yaml)

The `App-AutoScaler` provides the capability to adjust the computation resources for Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The `App-AutoScaler` has the following components:

* `api` : provides public APIs to manage scaling policy
* `servicebroker`: implements the [Cloud Foundry service broker API][k]
* `metricsgateway` : collects and filters loggregator events via loggregator v2  API
* `metricsserver`: transforms loggregator events to app-autoscaler performance metrics ( metricsgateway + metricsserver is a replacement of metricscollector)
* `metricsforwarder`: receives and forwards custom metrics to loggreator via v2 ingress API
* `eventgenerator`: aggreates memory metrics, evaluates scaling rules and triggers events for dynamic scaling
* `scheduler`: manages the schedules in scaling policy and trigger events for scheduled scaling
* `scalingengine`: takes the scaling actions based on dynamic scaling rules or schedules

You can follow the development progress on [Pivotal Tracker][t].

## Development
 
### System requirements

* Java 8 or above
* [Apache Maven][b] 3
* [Cloud Foundry cf command line][f]
* Go 1.15 or above

### Database requirement

The `App-AutoScaler` supports Postgres and MySQL. It uses Postgres as the default backend data store. To download and install, refer to [PostgreSQL][p] and [MySQL][m] web site.


### Setup

To set up the development, firstly clone this project

```shell
$ git clone https://github.com/cloudfoundry/app-autoscaler.git
$ cd app-autoscaler
```


#### Initialize the Database

* **Postgres**

If you wish to use a docker image for psql then start it using:
```shell
docker run -p 5432:5432  --name postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=autoscaler -d postgres:9.6
```
and stop it using: ```docker rm -f postgres```

```shell
#Note if your using docker you do not need to do these 3 lines.
createuser postgres -s
psql postgres://postgres@127.0.0.1:5432 -c 'DROP DATABASE IF EXISTS autoscaler'
psql postgres://postgres@127.0.0.1:5432 -c 'CREATE DATABASE autoscaler'

mvn package -Dmaven.test.skip=true
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/api/db/api.db.changelog.yml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/servicebroker/db/servicebroker.db.changelog.json --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=scheduler/db/scheduler.changelog-master.yaml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=scheduler/db/quartz.changelog-master.yaml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/metricsserver/db/metricscollector.db.changelog.yml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/eventgenerator/db/dataaggregator.db.changelog.yml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/scalingengine/db/scalingengine.db.changelog.yml --username="postgres" --password="postgres" update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver --changeLogFile=src/autoscaler/operator/db/operator.db.changelog.yml --username="postgres" --password="postgres" update
```

* **MySQL**
```shell
mysql -u root -e "DROP DATABASE IF EXISTS autoscaler;"
mysql -u root -e "CREATE DATABASE autoscaler;"

mvn package
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/api/db/api.db.changelog.yml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/servicebroker/db/servicebroker.db.changelog.json --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=scheduler/db/scheduler.changelog-master.yaml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=scheduler/db/quartz.changelog-master.yaml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/metricsserver/db/metricscollector.db.changelog.yml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/eventgenerator/db/dataaggregator.db.changelog.yml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/scalingengine/db/scalingengine.db.changelog.yml --username=root update
java -cp 'db/target/lib/*' liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile=src/autoscaler/operator/db/operator.db.changelog.yml --username=root update
```
#### Generate TLS Certificates
create the certificates

**Note**: on macos it will install `certstrap` automatically but on other OS's it needs to be pre-installed
```shell
./scripts/generate_test_certs.sh
```


#### Install consul
To be able to run unit tests and integration tests, you'll need to install consul binary.
```
if uname -a | grep Darwin; then os=darwin; else os=linux; fi
curl -L -o $TMPDIR/consul-0.7.5.zip "https://releases.hashicorp.com/consul/0.7.5/consul_0.7.5_${os}_amd64.zip"
unzip $TMPDIR/consul-0.7.5.zip -d $GOPATH/bin
rm $TMPDIR/consul-0.7.5.zip
```

### Unit tests

* **Postgres**:
```shell
export DBURL=postgres://postgres@localhost/autoscaler?sslmode=disable
pushd src/autoscaler
  make buildtools
  make test
popd

mvn -pl scheduler test
```

* **MySQL**:
```shell
export DBURL="root@tcp(localhost)/autoscaler?tls=false"
make -C src/autoscaler buildtools
make -C src/autoscaler test

mvn test -pl scheduler -Dspring.profiles.active=mysql
```



### Integration tests

**Postgres**
```shell
mvn package -DskipTests
export DBURL=postgres://postgres@localhost/autoscaler?sslmode=disable
make -C src/autoscaler buildtools
make -C src/autoscaler integration
```

**MySQL**:
Just replace the $DBURL to `root@tcp(localhost)/autoscaler?tls=false`.

### Coding Standards
Autoscaler uses Golangci and Checkstyle for its code base. Refer to [style-guide](style-guide/README.md)

## Deploy and offer Auto-Scaler as a service

Go to [app-autoscaler-release][r] project for how to BOSH deploy `App-AutoScaler`

## Use Auto-Scaler service

Refer to [user guide][u] for the details of how to use the Auto-Scaler service, including policy definition, supported metrics, public API specification and command line tool.

## License

This project is released under version 2.0 of the [Apache License][l].


[b]: https://maven.apache.org/
[c]: http://couchdb.apache.org/
[d]: http://www.eclipse.org/m2e/
[e]: http://www.cloudant.com
[f]: https://github.com/cloudfoundry/cli/releases
[k]: http://docs.cloudfoundry.org/services/api.html
[l]: LICENSE
[t]: https://www.pivotaltracker.com/projects/1566795
[p]: https://www.postgresql.org/
[r]: https://github.com/cloudfoundry/app-autoscaler-release
[u]: docs/Readme.md
[m]: https://www.mysql.com/
