# AutoScaler UAT Guide

Tests that will NOT be introduced here are ones which could be tested at the component level.

NOTE: Because we want to parallelize execution, tests should be written in such a way as to be runnable individually. This means that tests should not depend on state in other tests,
and should not modify the CF state in such a way as to impact other tests.

1. [Test Setup](#test-setup)
    1. [Install Required Dependencies](#install-required-dependencies)
    1. [Test Configuration](#test-configuration)
1. [Test Execution](#test-execution)

## Test Setup

### Install Required Dependencies

Set up your golang development environment, per [golang.org](http://golang.org/doc/install).

See [cf CLI](https://github.com/cloudfoundry/cli) for instructions on installing the go version of `cf`. The latest CF CLI version are recommended.

Make sure that the go version of `cf` is accessible in your `$PATH`.

You will also need a running Cloud Foundry deployment with the AutoScaler installed to run these acceptance tests against.

### Test Configuration

You must set an environment variable `$CONFIG` which points to a JSON file that contains several pieces of data that will be used to configure the acceptance tests, e.g. telling the tests how to target your running Cloud Foundry deployment.

The following can be pasted into a terminal and will set up a sufficient `$CONFIG` to run the core test suites against a [BOSH-Lite](https://github.com/cloudfoundry/bosh-lite) deployment of CF.

```bash
cat > integration_config.json <<EOF
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,

  "service_name": "autoscaler",
  "service_plan": "autoscaler-free-plan",
  "aggregate_interval": 120,
  "health_endpoints_basic_auth_enabled": false,

  "autoscaler_api": "autoscaler.bosh-lite.com",
}
EOF
export CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:

| Parameter                                           | Optionality |      Default       | Description                                                                                                                                                                                                                                                                                                                  |
|:----------------------------------------------------|:-----------:|:------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **service_name**                                    |  required   |                    | The name of the registered auto-scaler service, use `cf marketplace` to determine the name.                                                                                                                                                                                                                                  |
| **service_plan**                                    |  required   |                    | The plan name of the registered auto-scaler service, use `cf marketplace` to determine the plan.                                                                                                                                                                                                                             |
| **aggregate_interval**                              |  required   |                    | How frequently metrics are aggregated. This value must match the value configured in your deployment.                                                                                                                                                                                                                        |
| **autoscaler_api**                                  |  required   |                    | AutoScaler API endpoint                                                                                                                                                                                                                                                                                                      |
| **api**                                             |  required   |                    | Cloud Controller API endpoint.                                                                                                                                                                                                                                                                                               |
| **admin_user**                                      |  required   |                    | Name of a user in your CF instance with admin credentials.  This admin user must have the `doppler.firehose` scope if running the `logging` firehose tests.                                                                                                                                                                  |
| **admin_password**                                  |  required   |                    | Password of the admin user above.                                                                                                                                                                                                                                                                                            |
| **apps_domain**                                     |  required   |                    | A shared domain that tests can use to create subdomains that will route to applications also craeted in the tests.                                                                                                                                                                                                           |
| **health_endpoints_basic_auth_enabled**             |  optional   |        true        | Set to false if you did not enable basic auth on the health endpoints. Default is true.                                                                                                                                                                                                                                      |
| **enable_service_access**                           |  optional   |        true        | Set to false if autoscaler is offered as a cloudfoundry service which is globally enabled. Default is true.                                                                                                                                                                                                                  |
| **skip_ssl_validation**                             |  optional   |       false        | Set to true if using an invalid (e.g. self-signed) cert for traffic routed to your CF instance; this is generally always true for BOSH-Lite deployments of CF.                                                                                                                                                               |
| **use_existing_user**                               |  optional   |       false        | The admin user configured above will normally be used to create a temporary user (with lesser permissions) to perform actions (such as push applications) during tests, and then delete said user after the tests have run; set this to `true` if you want to use an existing user, configured via the following properties. |
| **keep_user_at_suite_end**                          |  optional   |       false        | If using an existing user (see above), set this to `true` unless you are okay having your existing user being deleted at the end. You can also set this to `true` when not using an existing user if you want to leave the temporary user around for debugging purposes after the test teardown.                             |
| **existing_user**                                   |  optional   |         ""         | Name of the existing user to use.                                                                                                                                                                                                                                                                                            |
| **existing_user_password**                          |  optional   |         ""         | Password for the existing user to use.                                                                                                                                                                                                                                                                                       |
| **artifacts_directory**                             |  optional   |    "../results"    | If set, `cf` CLI trace output from test runs will be captured in files and placed in this directory. [See below](#capturing-test-output) for more.                                                                                                                                                                           |
| **default_timeout**                                 |  optional   |         30         | Default time (in seconds) to wait for polling assertions that wait for asynchronous results.                                                                                                                                                                                                                                 |
| **cf_push_timeout**                                 |  optional   |         3          | Default time (in minutes) to wait for `cf push` commands to succeed.                                                                                                                                                                                                                                                         |
| **long_curl_timeout**                               |  optional   |         2          | Default time (in minutes) to wait for assertions that `curl` slow endpoints of test applications.                                                                                                                                                                                                                            |
| **test_password**                                   |  optional   |         ""         | Used to set the password for the test user. This may be needed if your CF installation has password policies.                                                                                                                                                                                                                |
| **timeout_scale**                                   |  optional   |        1.0         | Used primarily to scale default timeouts for test setup and teardown actions (e.g. creating an org) as opposed to main test actions (e.g. pushing an app).                                                                                                                                                                   |
| **use_http**                                        |  optional   |       false        | Set to true if you would like CF Acceptance Tests to use HTTP when making api and application requests. (default is HTTPS)                                                                                                                                                                                                   |
| **node_memory_limit**                               |  optional   |        128         | Default memory limit (in MB) of node.js test application, should be greater than 128 (MB).This is currently useful for the cpu quota given that directly relates to the memory given in some environments.                                                                                                                   |
| **binary_buildpack_name**                           |  optional   | "binary_buildpack" | [See below](#buildpack-names).                                                                                                                                                                                                                                                                                               |
| **cpu_upper_threshold**                             |  optional   |        100         |                                                                                                                                                                                                                                                                                                                              |
| **broker_start_timeout**                            |  optional   |         5          | Default time (in minutes)                                                                                                                                                                                                                                                                                                    |
| **async_service_operation_timeout**                 |  optional   |         2          | Default time (in minutes)                                                                                                                                                                                                                                                                                                    |
| **detect_timeout**                                  |  optional   |         5          | Default time (in minutes)                                                                                                                                                                                                                                                                                                    |
| **sleep_timeout**                                   |  optional   |         30         | Default time (in seconds)                                                                                                                                                                                                                                                                                                    |
| **name_prefix**                                     |  optional   |      "ASATS"       |                                                                                                                                                                                                                                                                                                                              |
| **instance_prefix**                                 |  optional   |     "service"      |                                                                                                                                                                                                                                                                                                                              |
| **app_prefix**                                      |  optional   |     "nodeapp"      |                                                                                                                                                                                                                                                                                                                              |
| **prefix**                                          |  optional   |    "autoscaler"    |                                                                                                                                                                                                                                                                                                                              |
| **service_broker**                                  |  optional   |    "autoscaler"    |                                                                                                                                                                                                                                                                                                                              |
| **cf_java_timeout**                                 |  optional   |         10         | Default time (in minutes)                                                                                                                                                                                                                                                                                                    |
| **use_existing_organization**                       |  optional   |       false        |                                                                                                                                                                                                                                                                                                                              |
| **existing_organization**                           |  optional   |         ""         |                                                                                                                                                                                                                                                                                                                              |
| **add_existing_user_to_existing_space**             |  optional   |       false        |                                                                                                                                                                                                                                                                                                                              |
| **use_existing_space**                              |  optional   |       false        |                                                                                                                                                                                                                                                                                                                              |
| **cpuutil_scaling_policy_test.app_cpu_entitlement** |  optional   |         25         |                                                                                                                                                                                                                                                                                                                              |
| **cpuutil_scaling_policy_test.app_memory**          |  optional   |       "1GB"        |                                                                                                                                                                                                                                                                                                                              |


#### Buildpack Names
Many tests specify the buildpack when pushing an app, so that the app staging process completes faster. The default name for the buildpack is as follows; if you have a "binary buildpack" with a different name, you can override it by setting a different name:

* `binary_buildpack_name: binary_buildpack`

#### Capturing Test Output
If you set a value for `artifacts_directory` in your `$CONFIG` file, then you will be able to capture `cf` trace output from failed test runs.  When a test fails, look for the node id and suite name ("*Applications*" and "*2*" in the example below) in the test output:

```bash
=== RUN TestLifecycle

Running Suite: Applications
====================================
Random Seed: 1389376383
Parallel test node 2/10. Assigned 14 of 137 specs.
```

The `cf` trace output for the tests in these specs will be found in `CF-TRACE-Applications-2.txt` in the `artifacts_directory`.

### Test Execution

Before executing the tests, the compiled test app needs to be available to the test suite.
If you are running the test suite from the checked out repository you can compile the test app by running

```bash
make build-test-app
```

Alternatively, you can use the acceptance test package that is released in conjunction with the BOSH releases and can be found on the [GitHub releases page](https://github.com/cloudfoundry/app-autoscaler-release/releases/tag/v10.0.3),e.g., https://github.com/cloudfoundry/app-autoscaler-release/releases/download/v10.0.3/app-autoscaler-acceptance-tests-v10.0.3.tgz. This package only contains the acceptance test suite and the pre-compiled test app.

There are several different test suites, and you may not wish to run all the tests in all contexts, and sometimes you may want to focus individual test suites to pinpoint a failure.  The default set of tests can be run via:

```bash
./bin/test_default
```

For more flexibility you can run `./bin/test` and specify many more options, e.g. which suites to run, which suites to exclude (e.g. if you want to run all but one suite), whether or not to run the tests in parallel, the number of parallel nodes to use, etc.  Refer to [ginkgo documentation](http://onsi.github.io/ginkgo/) for full details.

For example, to execute all test suites, and have tests run in parallel across four processes one would run:

```bash
./bin/test -r --nodes=4 --flake-attempts=3
```

*NOTE*:
 -  *--nodes*: indicates how many tests to run in parrallel. We have found these to be flaky and probably not used for a stable run.
 -  *--flake-attempts*: This will help with the flakiness of some tests. We are trying to stablise them better but this will take a long time.

Be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.
To execute the acceptance tests for a specific suite, e.g. `broker`, run the following:

```bash
./bin/test broker
```

The suite names correspond to directory names.

To see verbose output from `ginkgo`, use the `-v` or '-vv' flag.

```bash
./bin/test broker -v
```

Most of these flags and options can also be passed to the `bin/test_default` scripts as well.
