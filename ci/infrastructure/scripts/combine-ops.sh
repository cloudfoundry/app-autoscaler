#!/bin/bash

set -euo pipefail

mkdir -p combined-ops/operations/cf combined-ops/operations/autoscaler

cp -r cf-deployment/operations/* combined-ops/operations/cf/
cp -r ci/ci/operations/* combined-ops/operations/autoscaler
