<link href="https://raw.github.com/clownfart/Markdown-CSS/master/markdown.css" rel="stylesheet"></link>
# CF-AutoScaler

This is an incubation project for Cloud Foundry. You can follow the development progress on [Pivotal Tracker][t].

The `CF-AutoScaler` provides the capability to adjust the computation resources for CloudFoundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The `CF-AutoScaler` is provided as a Cloud Foundry service offering. Any application bound with `CF-AutoScaler` service will be able to use it. 

## Get Started

System requirements:

* Java 7 or above
* [Apache Maven][b] 3
* [Apache couchdb][c] 
* [CloudFoundry cf command line] [f]
* [CloudFoundry UAA command line client][u]

Database requirement:

The `CF-AutoScaler` uses Apache couchdb as the backend data store. You can have your own database installation from [Apache couchdb web site][c] or use an exisiting couchdb service, for example [Cloudant][e]


To get started, clone this project:

```shell
    $ git clone git://github.com/cfibmers/open-Autoscaler
    $ cd open-Autoscaler
```

The AutoScaler has three components, all of them them are Java Web Applications. 

* `api` : provides public APIs to manage scaling policy, retrive application metrics, and scaling history. See details in [API_usage.rst][a]
* `servicebroker`: implements the [Cloudfoundry service broker API][k]
* `server`: the backend engine of `CF-AutoScaler`

The following sections describe how to test, deploy and run `CF-AutoScaler` service manually. You can also use script `./bin/getStart.sh` to complete all these steps.


### Run Unit Test

For all these three `CF-AutoScaler` components, a unit test profile `unittest.properties` under 'profiles' directory has been created to define the settings for unit test.

To run unit test,  go to the project directory for each component, and launch mvn test. For example, to run unit test for service broker component:

```shell
cd servicebroker
mvn test -Punittest
```

### Configure and Package 


All the `CF-AutoScaler` components are configured through Maven profiles. You need to create your own profile according to the settings of the runtime environment that these components will be deployed to.

CloudFoundry settings

```
cfUrl=api.bosh-lite.com
cfClientId=cf-autoscaler-client
cfClientSecret=cf-autoscaler-client-secret
```

Service borker settings

```
service.name=CF-AutoScaler
brokerUsername=admin
brokerPassword=admin
```

Scaling Server URL and protocol settings 

```
serverURIList=AutoScaling.bosh-lite.com
apiServerURI=AutoScalingAPI.bosh-lite.com
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


Assuming you are using "sample" profile, add the profile to `~/.m2/settings.xml`, or `pom.xml` of each component project if it does not exist,  and then edit the properties in `{project}/profiles/sample.properties` for `api` , `servicebroker`, and `server`.


After the profiles are properly configured,  you can package the projects to .war files for deployment, using mvn as below. The *.war file can be found in folder `{project}/target`

```shell
mvn clean package -Psample -DskipTests
```

### Deploy `CF-AutoScaler` service

You can push the .war package of each `CF-AutoScaler` component to CloudFoundry to get `CF-AutoScaler` service deployed. Please note you need to use the URLs you configured in `servicebroker/profiles/{profile}.properties` as the routes of CF-AutoScaler server and API server. 

```shell
serverURIList=AutoScaling.bosh-lite.com
apiServerURI=AutoScalingAPI.bosh-lite.com
```

The deployment assumes that there is a couchdb available with the same host, port, username and password settings specified in the maven profile when packaging, and the couchdb can be accessed from the `CF-AutoScaler` running environment.

### Register `CF-AutoScaler` service broker


`CF-AutoScaler` invokes Cloud controller API to trigger scaling on target application. To achieve this, a UAA client id with  authorities `cloud_controller.read,cloud_controller.admin` is needed. You can create it using UAA command line client, make sure the client ID and secret are the ones you configured in the maven profile when you package the .war files

```shell
uaac target http://uaa.<cf-domain>
uaac token client get admin -s <cf uaa admin secret> 
uaac client add cf-autoscaler-client \
	--name cf-autoscaler \
    --authorized_grant_types client_credentials \
    --authorities cloud_controller.read,cloud_controller.admin \
    --secret cf-autoscaler-client-secret
```


Then you can register `CF-AutoScaler` with `cf` command line. Again make sure the service broker name, username and password are the ones you configured in the maven profile.

```shell
cf create-service-broker CF-AutoScaler brokerUserName brokerPassword brokerURI
cf enable-service-access CF-AutoScaler
```

## Test 'CF-AutoScaler' deployment

Run `bin/script/test/launchTest.sh` to test your deployements including service creat/delete/bind/unbind, scaling APIs, metrics and schedule based scaling using sample applications.

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

