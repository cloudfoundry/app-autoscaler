<link href="https://raw.github.com/clownfart/Markdown-CSS/master/markdown.css" rel="stylesheet"></link>
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
* [Apache couchdb][c]
* [Cloud Foundry cf command line] [f]
* [Cloud Foundry UAA command line client][u]

Database requirement:

The `CF-AutoScaler` uses Apache couchdb as the backend data store. You can have your own database installation from [Apache couchdb web site][c] or use an exisiting couchdb service, for example [Cloudant][e]


To get started, clone this project:

```shell
$ git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git
$ cd app-autoscaler
```

The AutoScaler has three components, all of them them are Java Web Applications.

* `api` : provides public APIs to manage scaling policy, retrive application metrics, and scaling history. See details in [API_usage.rst][a]
* `servicebroker`: implements the [Cloud Foundry service broker API][k]
* `server`: the backend engine of `CF-AutoScaler`



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

The following sections describe how to test, deploy and run `CF-AutoScaler` service manually. You can also use script `./bin/getStart.sh` to complete all these steps.

### Run Unit Tests

For all these three `CF-AutoScaler` components, a unit test `unittest.properties` under 'profiles' directory has been created to define the settings for unit test.

To run all the unit tests, launch mvn test.
```shell
mvn test -Denv=unittest
```

To run a specific project's unittest, launch mvn test for the project and its dependencies. For example, to run unit test for service broker component:
```shell
mvn test -Denv=unittest -pl common,servicebroker
```

### Configure and Package

All the `CF-AutoScaler` components are configured through a single properties file. You need to create your own properties according to the settings of your runtime environment.

Cloud Foundry settings

```
cfUrl=api.bosh-lite.com
cfClientId=cf-autoscaler-client
cfClientSecret=cf-autoscaler-client-secret
```

Service broker settings

```
service.name=CF-AutoScaler
brokerUsername=admin
brokerPassword=admin
```

Scaling Server URL and protocol settings

```
serverURIList=autoscaling.bosh-lite.com
apiServerURI=autoscalingapi.bosh-lite.com
httpProtocol=https
```

HTTP basic settings for authentication between CF-Autoscaler components

```
internalAuthUsername=admin
internalAuthPassword=admin
```

couchdb settings

```
couchdbUsername=
couchdbPassword=
couchdbHost=xxx.xxx.xxx.xxx
couchdbPort=5984
couchdbBrokerDBName=couchdb-scalingbroker
couchdbServerDBName=couchdb-scaling
couchdbMetricDBPrefix=couchdb-scalingmetric
```

metrics settings

```
reportInterval=120
```

Assume you want to name your environment, "myenv". Create a properties file in `app-autoscaler/profiles/sample.properties` with all the properties defined above.

To build all the .war files for deployment mvn as below. The *.war file can be found in each folder `{project}/target`

```shell
mvn clean package -Denv=sample -DskipTests
```

### Deploy `CF-AutoScaler` service

You can push the .war package of each `CF-AutoScaler` component to Cloud Foundry to get `CF-AutoScaler` service deployed. Please note you need to use the URLs you configured in `servicebroker/profiles/{profile}.properties` as the routes of CF-AutoScaler server and API server.

```shell
serverURIList=autoscaling.bosh-lite.com
apiServerURI=autoscalingapi.bosh-lite.com
```

The deployment assumes that there is a couchdb available with the same host, port, username and password settings specified in the maven profile when packaging, and the couchdb can be accessed from the `CF-AutoScaler` running environment.

### Register `CF-AutoScaler` service broker

You can register `CF-AutoScaler` with `cf` command line. Again make sure the service broker name, username and password are the ones you configured in the maven profile.

```shell
cf create-service-broker CF-AutoScaler brokerUserName brokerPassword brokerURI
cf enable-service-access CF-AutoScaler
```

## Test your 'CF-AutoScaler' deployment
See [src/acceptance/README.md](src/acceptance/README.md)

## Use `CF-AutoScaler`

Now, you can play with `CF-AutoScaler`.
Firstly create a `CF-AutoScaler` service, and bind to you application

``` shell
cf create-service CF-AutoScaler free <service_instance>
cf bind-service <app> <service_instance>
```

Then refer to [API_usage.rst][a] to manage the scaling policy of your application, retrieve metrics and scaling histories.


## License

This project is released under version 2.0 of the [Apache License][l].


[a]: https://github.com/cfibmers/open-Autoscaler/blob/master/docs/API_usage.rst
[b]: https://maven.apache.org/
[c]: http://couchdb.apache.org/
[d]: http://www.eclipse.org/m2e/
[e]: http://www.cloudant.com
[f]: https://github.com/cloudfoundry/cli/releases
[k]: http://docs.cloudfoundry.org/services/api.html
[l]: LICENSE
[t]: https://www.pivotaltracker.com/projects/1566795
[u]: https://github.com/cloudfoundry/cf-uaac
