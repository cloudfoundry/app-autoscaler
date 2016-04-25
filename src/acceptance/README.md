# AutoScaler Acceptance Tests

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
  "use_http": false,

  "service_name": "CF-AutoScaler",
  "api_url": "https://autoscalingapi.bosh-lite.com"
}
EOF
export CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:

* `service_name` (required): The name of the registered auto-scaler service, use `cf marketplace` to determine the name.
* `api_url` (required): The url of the API service of the auto-scaler

* `api` (required): Cloud Controller API endpoint.
* `admin_user` (required): Name of a user in your CF instance with admin credentials.  This admin user must have the `doppler.firehose` scope if running the `logging` firehose tests.
* `admin_password` (required): Password of the admin user above.
* `apps_domain` (required): A shared domain that tests can use to create subdomains that will route to applications also craeted in the tests.
* `skip_ssl_validation`: Set to true if using an invalid (e.g. self-signed) cert for traffic routed to your CF instance; this is generally always true for BOSH-Lite deployments of CF.
* `use_existing_user` (optional): The admin user configured above will normally be used to create a temporary user (with lesser permissions) to perform actions (such as push applications) during tests, and then delete said user after the tests have run; set this to `true` if you want to use an existing user, configured via the following properties.
* `keep_user_at_suite_end` (optional): If using an existing user (see above), set this to `true` unless you are okay having your existing user being deleted at the end. You can also set this to `true` when not using an existing user if you want to leave the temporary user around for debugging purposes after the test teardown.
* `existing_user` (optional): Name of the existing user to use.
* `existing_user_password` (optional): Password for the existing user to use.
* `backend` (optional): Set to 'diego' or 'dea' to determine the backend used. If unspecified the default backend will be used.
* `artifacts_directory` (optional): If set, `cf` CLI trace output from test runs will be captured in files and placed in this directory. [See below](#capturing-test-output) for more.
* `default_timeout` (optional): Default time (in seconds) to wait for polling assertions that wait for asynchronous results.
* `cf_push_timeout` (optional): Default time (in seconds) to wait for `cf push` commands to succeed.
* `long_curl_timeout` (optional): Default time (in seconds) to wait for assertions that `curl` slow endpoints of test applications.
* `test_password` (optional): Used to set the password for the test user. This may be needed if your CF installation has password policies.
* `timeout_scale` (optional): Used primarily to scale default timeouts for test setup and teardown actions (e.g. creating an org) as opposed to main test actions (e.g. pushing an app).
* `use_http` (optional): Set to true if you would like CF Acceptance Tests to use HTTP when making api and application requests. (default is HTTPS)

* `java_buildpack_name` (optional) [See below](#buildpack-names).

#### Buildpack Names
Many tests specify a buildpack when pushing an app, so that on diego the app staging process completes in less time. The default names for the buildpacks are as follows; if you have buildpacks with different names, you can override them by setting different names:

* `java_buildpack_name: java_buildpack`

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

There are several different test suites, and you may not wish to run all the tests in all contexts, and sometimes you may want to focus individual test suites to pinpoint a failure.  The default set of tests can be run via:

```bash
./bin/test_default
```

For more flexibility you can run `./bin/test` and specify many more options, e.g. which suites to run, which suites to exclude (e.g. if you want to run all but one suite), whether or not to run the tests in parallel, the number of parallel nodes to use, etc.  Refer to [ginkgo documentation](http://onsi.github.io/ginkgo/) for full details.  

For example, to execute all test suites, and have tests run in parallel across four processes one would run:

```bash
./bin/test -r -nodes=4
```

Be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.

To execute the acceptance tests for a specific suite, e.g. `api`, run the following:

```bash
bin/test api
```

The suite names correspond to directory names.

To see verbose output from `ginkgo`, use the `-v` flag.

```bash
./bin/test api -v
```

Most of these flags and options can also be passed to the `bin/test_default` scripts as well.
