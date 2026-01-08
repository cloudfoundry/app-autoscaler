#!/usr/bin/env bash
set -euo pipefail

# This script runs acceptance tests as a CloudFoundry task
# It expects the following environment variables:
# - ACCEPTANCE_CONFIG_JSON: JSON configuration (required)
# - SUITES: Space-separated list of test suites to run (default: "api app broker")
# - GINKGO_OPTS: Additional ginkgo options (optional)
# - NODES: Number of parallel nodes (default: 3)
# - SKIP_TEARDOWN: Skip cleanup after tests (default: false)

# Determine platform and architecture
GOOS=$(uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=$(uname -m)
case "$GOARCH" in
    x86_64) GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    armv7l) GOARCH="arm" ;;
    *) echo "Unsupported architecture: $GOARCH"; exit 1 ;;
esac

# Set defaults
SUITES="${SUITES:-}"
NODES="${NODES:-3}"
SKIP_TEARDOWN="${SKIP_TEARDOWN:-false}"
GINKGO_OPTS="${GINKGO_OPTS:-}"

# Exit early if no suites specified (skip tests)
if [ -z "$SUITES" ]; then
    echo "No test suites specified (SUITES is empty)"
    echo "Skipping acceptance tests"
    exit 0
fi

# Validate required environment variables
if [ -z "${ACCEPTANCE_CONFIG_JSON:-}" ]; then
    echo "ERROR: ACCEPTANCE_CONFIG_JSON environment variable is required"
    echo "It should contain the full JSON configuration for the acceptance tests"
    exit 1
fi

# Set working directory (already in /home/vcap/app with test files)
cd /home/vcap/app

# Add bundled CF CLI to PATH
export PATH="/home/vcap/app/bin:$PATH"

# Verify CF CLI is available
if ! command -v cf &> /dev/null; then
    echo "ERROR: CF CLI not found in PATH"
    echo "Expected location: /home/vcap/app/bin/cf"
    ls -la /home/vcap/app/bin/ 2>/dev/null || echo "Directory /home/vcap/app/bin does not exist"
    exit 1
fi

echo "CF CLI found: $(cf version)"

# Set ginkgo binary path based on platform
GINKGO_BINARY="./ginkgo_v2_${GOOS}_${GOARCH}"
if [ ! -x "$GINKGO_BINARY" ]; then
    echo "ERROR: Ginkgo binary not found at $GINKGO_BINARY"
    echo "Expected platform: ${GOOS}_${GOARCH}"
    ls -la ginkgo_v2_* 2>/dev/null || echo "No ginkgo binaries found"
    exit 1
fi
export GINKGO_BINARY

# Build test suite list with platform-specific binaries
SUITE_ARGS=""
for suite in $SUITES; do
    suite_binary="${suite}/${suite}_${GOOS}_${GOARCH}.test"
    if [ ! -f "$suite_binary" ]; then
        echo "WARNING: Test suite binary not found: $suite_binary"
        echo "Available suites:"
        find . -name "*.test" -type f 2>/dev/null | sed 's|^\./||' || echo "  No test binaries found"
        continue
    fi
    SUITE_ARGS="$SUITE_ARGS $suite_binary"
done

if [ -z "$SUITE_ARGS" ]; then
    echo "ERROR: No valid test suites found to run"
    echo "Requested: $SUITES"
    exit 1
fi

echo "========================================"
echo "Running Acceptance Tests as CF Task"
echo "========================================"
echo "Platform: ${GOOS}_${GOARCH}"
echo "Suites: $SUITE_ARGS"
echo "Nodes: $NODES"
echo "Skip Teardown: $SKIP_TEARDOWN"
echo "========================================"

# Export environment variables for tests
export SKIP_TEARDOWN
export DEBUG=true

# Run the tests using the Ginkgo wrapper
# The config will be automatically picked up from ACCEPTANCE_CONFIG_JSON
# Test binaries are built with coverage support (ginkgo build --cover)
# shellcheck disable=SC2086
$GINKGO_BINARY -race -nodes="$NODES" -trace $GINKGO_OPTS $SUITE_ARGS

TEST_EXIT_CODE=$?

echo "========================================"
echo "Acceptance Tests Completed"
echo "Exit Code: $TEST_EXIT_CODE"
echo "========================================"

exit $TEST_EXIT_CODE
