# Application Autoscaler

The Application Autoscaler provides the capability to adjust the computation resources for Cloud Foundry applications through:

* Dynamic scaling based on application performance metrics
* Dynamic scaling based on custom metrics
* Scheduled scaling based on time

This repository contains the core Application Autoscaler source code, extracted and refactored from [app-autoscaler-release](https://github.com/cloudfoundry/app-autoscaler-release).

## Architecture

The Application Autoscaler consists of several microservices and are deployed as CF Applications

| Component         | Description                                                                 |
|-------------------|-----------------------------------------------------------------------------|
| `api`             | Public-facing API server for policy management and scaling history          |
| `servicebroker`   | Cloud Foundry service broker implementation                                 |
| `scheduler`       | Manages scheduled scaling policies and triggers scaling actions             |
| `eventgenerator`  | Evaluates scaling rules and generates scaling events based on metrics       |
| `scalingengine`   | Executes scaling decisions by adjusting application instances               |
| `metricsforwarder`| Forwards custom application metrics to the autoscaler                       |
| `operator`        | Manages autoscaler operations and instance synchronization                  |

## API Specifications

OpenAPI specifications are available in the [`openapi/`](./openapi/) directory:

* `application-metric-api.openapi.yaml` - Application metrics API
* `custom-metrics-api.openapi.yaml` - Custom metrics submission API
* `policy-api.openapi.yaml` - Scaling policy management API
* `scaling-history-api.openapi.yaml` - Scaling history query API
* `internal-scaling-history-api.openapi.yaml` - Internal scaling history API

## Local Development

### Prerequisites

* [Go](https://golang.org/) 1.24.3 or later
* [Docker](https://www.docker.com/products/docker-desktop/) to spin up required databases
* [devbox](https://github.com/jetify-com/devbox) (optional but recommended) - starts a shell with all required tools
* [Maven](https://maven.apache.org/) for building the Java-based scheduler component
* [direnv](https://direnv.net/) (optional) to automatically configure the development environment

### Make Targets

| Target                                                                | Description                                                                |
|-----------------------------------------------------------------------|----------------------------------------------------------------------------|
| `make build`                                                          | Build all components                                                       |
| `make generate-fakes`                                                 | Generate test mocks/fakes                                                  |
| `make generate-openapi-generated-clients-and-servers`                 | Generate clients and servers from OpenAPI specs                            |
| `make test`                                                           | Run unit tests against PostgreSQL                                          |
| `make clean && make test POSTGRES_TAG=x.y`                            | Run unit tests against specific PostgreSQL version                         |
| `make integration`                                                    | Run integration tests against PostgreSQL                                   |
| `make clean && make integration POSTGRES_TAG=x.y`                     | Run integration tests against specific PostgreSQL version                  |
| `make acceptance-tests`                                               | Run acceptance tests (see [acceptance/README.md](acceptance/README.md))    |
| `make lint`                                                           | Check code style                                                           |
| `OPTS=--fix make lint`                                                | Check code style and apply auto-fixes                                      |
| `make fmt`                                                            | Format Go code                                                             |
| `make clean`                                                          | Remove build artifacts and generated code                                  |
| `make mta-build`                                                      | Build MTA archive for deployment                                           |
| `make mta-deploy`                                                     | Deploy to Cloud Foundry using MTA                                          |


## Use Application Autoscaler Service

Refer to [`user guide`](./docs/user_guide.md) for the details of how to use the Auto-Scaler service, including policy definition, supported metrics, public API specification and command line tool.

## Monitor Microservices

The app-autoscaler provides a number of health endpoints that are available externally that can be used to check the
state of each component. Each health endpoint is protected with basic auth (apart from the api server), the usernames
are listed in the table below, but the passwords are available in credhub.

| Component        | Health URL                                                   | Username         | Password Key                                 |
|------------------|--------------------------------------------------------------|------------------|----------------------------------------------|
| eventgenerator   | https://autoscaler-eventgenerator.((system_domain))/health   | eventgenerator   | /autoscaler_eventgenerator_health_password   |
| metricsforwarder | https://autoscaler-metricsforwarder.((system_domain))/health | metricsforwarder | /autoscaler_metricsforwarder_health_password |
| scalingengine    | https://autoscaler-scalingengine.((system_domain))/health    | scalingengine    | /autoscaler_scalingengine_health_password    |
| operator         | https://autoscaler-operator.((system_domain))/health         | operator         | /autoscaler_operator_health_password         |
| scheduler        | https://autoscaler-scheduler.((system_domain))/health        | scheduler        | /autoscaler_scheduler_health_password        |

### Running Tests

```bash
# Run all unit tests
make test

# Run integration tests
make integration

# Run acceptance tests (requires deployed autoscaler)
make acceptance-tests
```

Note: Running tests will automatically spin up the PostgreSQL database via Docker.

### Database Setup

The autoscaler supports PostgreSQL.

To manually start a local database for development:

```bash
make start-db                    # Start PostgreSQL
```

To stop the database:

```bash
make stop-db                     # Stop PostgreSQL
```

## Documentation

* [Acceptance Tests](acceptance/README.md) - Guide for running acceptance tests
* [Scheduler Component](scheduler/README.md) - Scheduler-specific documentation
* [OpenAPI Specifications](openapi/) - API documentation and schemas

## Related Repositories

* [app-autoscaler-release](https://github.com/cloudfoundry/app-autoscaler-release) - BOSH release for deploying the Application Autoscaler

## License

This project is released under version 2.0 of the [Apache License](LICENSE).
