<link href="https://raw.github.com/clownfart/Markdown-CSS/master/markdown.css" rel="stylesheet"></link>
# CloudFoundry AutoScaler

The `CF-AutoScaler` provides the capability to adjust the computation resources for CloudFoundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The `CF-AutoScaler` is provided as a Cloud Foundry service offering. Any application bound with `CF-AutoScaler` service will be able to use it. 

## Quick Start

System requirements:

* Java 7 or above
* [Apache Maven][b] 3
* [Apache couchdb][c] 
* cf command line version 6 

Database requirement:

The `CF-AutoScaler` uses Apache couchdb as the backend data store. You can have your own database installation from [here][c] or use an exisiting couchdb service, for example [Cloudant][e]


Requirements of Cloudfoudry access: 

* To register `CF-AutoScaler` service, you need a valid Cloudfoundry UAA user id.
* `CF-AutoScaler` will invoke Cloud controller API to trigger scaling on target application. To achieve this, a UAA client id with  authorities `cloud_controller.read,cloud_controller.admin` is needed. You can create it using UAA command line tool:

```shell
uaac target http://uaa.<cf-domain>
uaac token client get admin -s <cf uaa admin secret> 
uaac client add cf-autoscaler-client \
    --authorized_grant_types client_credentials \
    --authorities cloud_controller.read,cloud_controller.admin \
    --secret cf-autoscaler-client-secret
```

Then you can start with cloning this project:

```shell
    $ git clone git://github.com/cfibmers/open-Autoscaler
    $ cd open-Autoscaler
```
The `CF-AutoScaler` is offered as an Java Web Application. You need the following steps to get it running. 

* Package `CF-AutoScaler` with Maven
* Launch `CF-AutoScaler` runtime 
* Register `CF-AutoScaler` service broker on CloudFoundry

The script `./bin/getStart.sh` will help you to complete all these steps.  The manual guide is also listed as below.

### Package with Maven

The AutoScaler has three components, all of them are Java Web Application: 

* `api` : provides public APIs to manage scaling policy, retrive application metrics and scaling history. See details in [API_usage.rst][a]
* `servicebroker`: implements the [Cloudfoundry service broker API] (http://docs.cloudfoundry.org/services/api.html) to offer `CF-AutoScaler` as a service.
* `server`: the backend engine of `CF-AutoScaler`

All these 3 components are configured through Maven profiles. You need to create your own profile according to your runtime environment, and then package the projects to .war file. Here is an example:

* Assuming you are using "sample" profile, edit the properties in {project}/profiles/sample.properties for `api` , `servicebroker`, `server`
* Run `mvn clean package` to create *.war file which would be found in folder {project}/target

### Launch Runtime
As an Java web application, you can launch the components of `CF-AutoScaler` with Tomcat directly or push to CloudFoundry.

Please note the runtime environment must use the below settings you configured in `servicebroker`/profiles/{profile}.properties.

```shell
serverURIList=AutoScaling.bosh-lite.com
apiServerURI=AutoScalingAPI.bosh-lite.com
```
If you launch `CF-AutoScaler` with Tomcat server in Eclispe & plugin [M2eclipse][d], the following steps will help you to enable your customized configuration :

```shell
1. Run mvn eclipse:eclipse -Dwtpversion=2.0 for each projects
2. Activate <profile> in eclipse by Right-click the project -> Properties -> Maven -> Fill in the <profile> name you want to activate.
```

### Register `CF-AutoScaler` service broker
You can register `CF-AutoScaler` with command:

```shell
cf create-service-broker CF-AutoScaler brokerUserName brokerPassword brokerURI
cf enable-service-access CF-AutoScaler
```

### Use `CF-AutoScaler` 
Now, you can play with `CF-AutoScaler`.

Firstly create a `CF-AutoScaler` service, and bind to you application

``` shell
cf create-service `CF-AutoScaler` free <service_instance>
cf bind-service <app> <service_instance>
```

Then define scaling policy using the APIs described in [API_usage.rst][a]

## License

This project is released under version 2.0 of the [Apache License][l].
[a]: https://github.com/cfibmers/open-Autoscaler/blob/master/docs/API_usage.rst
[b]: https://maven.apache.org/
[c]: http://couchdb.apache.org/
[d]: http://www.eclipse.org/m2e/
[e]: http://www.cloudant.com
[l]: https://www.apache.org/licenses/LICENSE-2.0

