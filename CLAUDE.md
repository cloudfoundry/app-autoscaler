# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Application Autoscaler for Cloud Foundry** is a production-grade microservices system that automatically scales Cloud Foundry applications based on metrics (CPU, memory, throughput, custom metrics) and schedules. It acts as a Cloud Foundry service broker, allowing apps to bind to the autoscaler service and define scaling policies.

## Technology Stack

- **Go 1.24.3**: Used for 6 of 7 microservices (api, eventgenerator, scalingengine, metricsforwarder, operator, acceptance tests)
- **Java 17+/Spring Boot 3.5**: Used only for the Scheduler component with Quartz scheduler
- **PostgreSQL**: Primary database (multiple logical databases per component)
- **Testing**: Ginkgo/Gomega for Go, JUnit for Java
- **Mocking**: Counterfeiter for generating test fakes

## Build & Test Commands

### Building
```bash
make build                    # Build all Go services
make scheduler.build          # Build Java scheduler component
make build-test-app          # Build test application
make build_all               # Build everything (programs + tests)
make mta-build               # Build MTA archive for Cloud Foundry deployment
```

### Testing
```bash
make test                    # Run all unit tests (auto-starts PostgreSQL via Docker)
make autoscaler.test         # Run Go unit tests only
make scheduler.test          # Run Java scheduler tests
make integration             # Run integration tests
make acceptance-tests        # Run acceptance tests (requires deployed autoscaler)
make mta-acceptance-tests    # Run acceptance tests via CF tasks in parallel

# Run specific test suite
make test TEST=./api         # Run tests for specific package

# Test options via GINKGO_OPTS
GINKGO_OPTS="--focus=scaling" make test
```

### Database Management
```bash
make start-db                # Start PostgreSQL via Docker
make stop-db                 # Stop PostgreSQL
make init-db                 # Initialize database schemas
```

### Code Quality
```bash
make lint                    # Run all linters (Go, markdown, GitHub Actions)
OPTS=--fix make lint         # Auto-fix linting issues
make fmt                     # Format Go code
make generate-fakes          # Generate test mocks with Counterfeiter
make generate-openapi-generated-clients-and-servers  # Generate OpenAPI clients/servers
```

### Deployment
```bash
make mta-deploy              # Deploy to Cloud Foundry using MTA
make mta-undeploy            # Undeploy from Cloud Foundry
make deploy-register-cf      # Register service broker with CF
make deploy-cleanup          # Clean up autoscaler deployment
```

### Cleanup
```bash
make clean                   # Clean all build artifacts, fakes, and caches
make clean-acceptance        # Clean acceptance test artifacts
```

## High-Level Architecture

The autoscaler is a **distributed microservices architecture** with 7 components:

### Core Components

1. **API Server** (`/api`)
   - Public REST API and Cloud Foundry service broker implementation
   - Manages policies, bindings, scaling history queries
   - Uses: PolicyDB, BindingDB
   - Entry point: `api/cmd/api/main.go`

2. **Event Generator** (`/eventgenerator`)
   - Polls metrics from CF Log Cache, aggregates them, evaluates scaling rules
   - Triggers scaling events when thresholds breach
   - Components: MetricPoller, Aggregator, Evaluator, AppManager
   - Uses: AppMetricsDB, PolicyDB
   - Entry point: `eventgenerator/cmd/eventgenerator/main.go`

3. **Scaling Engine** (`/scalingengine`)
   - Executes scaling decisions by calling CF API
   - Manages cooldown periods to prevent oscillation
   - Records scaling history and synchronizes schedules
   - Uses: PolicyDB, ScalingEngineDB, SchedulerDB
   - Entry point: `scalingengine/cmd/scalingengine/main.go`

4. **Scheduler** (`/scheduler`) - **Java/Spring Boot**
   - Manages scheduled scaling (recurring/specific dates) using Quartz
   - **Only component written in Java**
   - Uses: SchedulerDB
   - Build: Maven with `pom.xml`

5. **Metrics Forwarder** (`/metricsforwarder`)
   - Receives custom app metrics via HTTP, forwards to Log Cache
   - Rate limits metric submissions
   - Uses: PolicyDB, BindingDB
   - Entry point: `metricsforwarder/cmd/metricsforwarder/main.go`

6. **Operator** (`/operator`)
   - Background tasks: database pruning, schedule sync, app state sync
   - Uses distributed locking for single active instance
   - Uses: AppMetricsDB, ScalingEngineDB, PolicyDB, LockDB
   - Entry point: `operator/cmd/operator/main.go`

7. **Service Broker** (part of API Server)
   - Cloud Foundry service broker API implementation
   - Handles provisioning, binding, policy management

### Data Flow Example

1. User creates service binding with policy → API Server → PolicyDB
2. Event Generator polls metrics from Log Cache → aggregates → evaluates against policy
3. Threshold breached → Event Generator calls Scaling Engine
4. Scaling Engine checks cooldown → scales via CF API → records history

## Important Development Patterns

### Shared Code Structure
- `/models`: Shared data models (policy, metrics, API types)
- `/db`: Database interfaces and SQL implementations
- `/cf`: Cloud Foundry client wrapper

### Database Architecture

Each service uses its own logical PostgreSQL database:

- `policy_db`: Scaling policies and credentials
- `binding_db`: Service bindings and instances
- `appmetrics_db`: Metrics time-series data
- `scalingengine_db`: Scaling history and cooldown state
- `scheduler_db`: Schedules and Quartz job data
- `lock_db`: Distributed locks

### Testing Patterns

- **Unit tests**: Ginkgo/Gomega framework (run with `make test`)
- **Fakes**: Generated via Counterfeiter (`make generate-fakes`)
- **Integration tests**: Require PostgreSQL (auto-started by tests)
- **Test certificates**: Auto-generated via `make test-certs`
- **Important**: Regenerate fakes after any interface changes

### Configuration Management

Services use YAML configuration files and environment variables.

**Key environment variables** (see `scripts/vars.source.sh`):
- `DEPLOYMENT_NAME`: Deployment identifier (default: `autoscaler-mta-${PR_NUMBER}`)
- `SYSTEM_DOMAIN`: CF system domain
- `DBURL`: Database connection string
- `BBL_STATE_PATH`: BBL state directory (optional, falls back to error message if missing)

### Scripts

**Critical scripts** in `/scripts`:
- `vars.source.sh`: Standard environment variables (source in other scripts)
- `mta-build.sh`: Builds MTA archive
- `mta-deploy.sh`: Deploys to Cloud Foundry
- `run-mta-acceptance-tests.sh`: Runs acceptance tests in parallel via CF tasks

**Note**: `vars.source.sh` uses ERR trap for error handling. Commands that fail will trigger error reporting.

## API Specifications

OpenAPI 3.0 specs in `/openapi`:
- `policy-api.openapi.yaml`: Scaling policy management
- `scaling-history-api.openapi.yaml`: Query scaling history
- `custom-metrics-api.openapi.yaml`: Submit custom metrics
- Code generation: `ogen` tool generates clients/servers

## Common Gotchas

1. **Counterfeiter**: Uses pinned `golang.org/x/tools v0.39.0` due to compatibility issues. Don't upgrade without checking.
2. **Database vendoring**: Run `make init-db` after database schema changes.
3. **Java component**: Scheduler is the only Java component, built separately with Maven.
4. **Test database**: Tests auto-start PostgreSQL via Docker. Use `db_type=postgres` (default) or `db_type=mysql`.
5. **MTA deployment**: Requires Cloud Foundry MTA plugin and proper credentials.
6. **Go modules**: `GOWORK=off` is set to disable workspace mode.
7. **FIPS builds**: `GOFIPS140=v1.0.0` enables FIPS 140 compliance.
8. **BBL_STATE_PATH**: Script continues gracefully if path doesn't exist (shows error message but doesn't fail).

## Acceptance Tests

Located in `/acceptance`:
- Require deployed autoscaler instance
- Configuration: `acceptance/acceptance_config.json`
- Can run in parallel via CF tasks: `make mta-acceptance-tests`
- See `acceptance/README.md` for details

## Team Workflows

### PR Comment Resolution Workflow

Standard process for addressing PR review comments:

1. Implement the fix and create a commit
2. Reply to the comment with: `fixed in <commit_hash>` (example: `fixed in 8384249f`)
3. Resolve the comment thread on GitHub

**Using GitHub CLI**:

```bash
# Capture the commit hash
COMMIT_HASH=$(git rev-parse --short HEAD)

# Get thread ID using /unresolved-comments skill
THREAD_ID="PRRT_kwDOA0tdb85qnILg"

# Reply to the comment
gh api graphql -f query='
mutation {
  addPullRequestReviewThreadReply(input: {
    pullRequestReviewThreadId: "'"${THREAD_ID}"'"
    body: "fixed in '"${COMMIT_HASH}"'"
  }) {
    comment { id }
  }
}'

# Resolve the thread
gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "'"${THREAD_ID}"'"}) {
    thread { isResolved }
  }
}'
```

**Finding thread IDs**: Use `/unresolved-comments` skill or query manually:

```bash
gh api graphql -f query='
query {
  repository(owner: "cloudfoundry", name: "app-autoscaler") {
    pullRequest(number: PR_NUMBER) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              path
              line
              body
            }
          }
        }
      }
    }
  }
}'
```
