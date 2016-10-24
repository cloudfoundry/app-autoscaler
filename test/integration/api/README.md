# api Integration Tests

#### Get Started with the Integration tests by cloning the repository
```sh
$ git clone https://github.com/cloudfoundry-incubator/app-autoscaler.git
$ cd app-autoscaler
```

#### Prerequisites
* Refer to System pre-requisites https://github.com/cloudfoundry-incubator/app-autoscaler#getting-started
* Setup the DB https://github.com/cloudfoundry-incubator/app-autoscaler#initialize-the-database
* Modify the DB_URI and the SCHEDULER_URI within the integration script in package.json to appropriate values (if required)

#### Environment Setup


```sh
pushd  scheduler
mvn package -DskipTests
popd

pushd api
npm install
popd

pushd test/integration/api
npm install
popd
```

#### Run Tests

```sh
pushd test/integration/api
npm run integration
```
