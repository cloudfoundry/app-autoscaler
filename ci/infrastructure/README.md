# Infrastructure Pipeline

Concourse CI pipeline for managing the App Autoscaler test infrastructure on GCP.

## Overview

This pipeline provisions and manages a complete Cloud Foundry environment with PostgreSQL databases for testing App Autoscaler. It uses [BBL (Bosh BootLoader)](https://github.com/cloudfoundry/bosh-bootloader) to provision infrastructure on GCP and deploys CF using [cf-deployment](https://github.com/cloudfoundry/cf-deployment).

**Environment**: `autoscaler.app-runtime-interfaces.ci.cloudfoundry.org`  
**Location**: GCP project `app-runtime-interfaces-wg`, region `europe-west3-a`

## Running the Pipeline

### Prerequisites

First, log in to the Concourse target:

```bash
fly login -t app-autoscaler-release \
  -c https://concourse.app-runtime-interfaces.ci.cloudfoundry.org \
  -n app-autoscaler
```

### Set the Pipeline

```bash
# From the ci directory (recommended)
cd ci
TARGET=app-autoscaler-release make set-infrastructure-pipeline

# Or using fly directly
fly -t app-autoscaler-release set-pipeline -p infrastructure \
  -c infrastructure/pipeline.yml
```

## Troubleshooting


### Fix Expired BOSH/Jumpbox Certificates

If BOSH Director or Jumpbox certificates expire (x509 certificate errors), see the [app-autoscaler-env-bbl-state README](https://github.com/cloudfoundry/app-autoscaler-env-bbl-state) for instructions on using the `remove-cert.sh` script to remove expired certificates from the vars stores.

After removing certificates, trigger `setup-infrastructure` to regenerate them:
```bash
fly -t app-autoscaler-release trigger-job -j infrastructure/setup-infrastructure
```

### Fix Expired CF Certificates (NATS, Loggregator, Metrics Scraper)

The `deploy-cf` job's `REGENERATE_CREDENTIALS: "true"` setting instructs BOSH to regenerate any missing certificates in CredHub. This regeneration is implemented by [cf-deployment-concourse-tasks/bosh-deploy](https://github.com/cloudfoundry/cf-deployment-concourse-tasks/blob/f5e2b103800315eac9fa6ba3e7b6c6054e17816c/bosh-deploy/task.yml#L58), which passes `--recreate` to `bosh deploy` when credentials need regeneration.


```
# example log
nats/3043ea83-fb05-41b3-8e2b-e59acd53fd0c: stdout | [main] 2026-07-10T10:43:43.537816861Z ERROR - Unable to configure health checker failed to load keypair: certificate has expired: validity ended at 2026-07-09 11:21:32 UTC but current time is 2026-07-10 10:43:43 UTC
```

1. Ensure `REGENERATE_CREDENTIALS: "true"` is set in the pipeline
2. Set the pipeline: `cd ci && TARGET=app-autoscaler-release make set-infrastructure-pipeline`
3. Trigger the CF deployment to regenerate certificates:
   ```bash
   fly -t app-autoscaler-release trigger-job -j infrastructure/deploy-cf
   ```
