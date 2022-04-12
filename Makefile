SHELL := /bin/bash
GO_VERSION := $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')
PACKAGE_DIRS := $(shell go list ./... | grep -v /vendor/ | grep -v e2e)
CGO_ENABLED = 0
BUILDTAGS :=
export GO111MODULE=on
# Fix for golang issue with Montery 6/12/2021. Fix already in trunk but not released
# https://github.com/golang/go/issues/49138
# https://github.com/golang/go/commit/5f6552018d1ec920c3ca3d459691528f48363c3c
export MallocNanoZone=0

#TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/564 allow the tests to be run in parallel
#GINKGO_OPTS=-r --race --require-suite -p --randomize-all --cover

GINKGO_OPTS=-r --race --require-suite --randomize-all --cover

build-%:
	@echo "# building $*"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go

build:	build-scalingengine\
		build-metricsforwarder\
		build-eventgenerator\
		build-api\
		build-metricsgateway\
		build-metricsserver\
		build-operator\

check: fmt lint build test

test:
	@echo "Running tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo ${GINKGO_OPTS} --skip-package=integration

testsuite:
	APP_AUTOSCALER_TEST_RUN=true ginkgo ${GINKGO_OPTS}  $(TEST)

.PHONY: integration
integration:
	@echo "# Running integration tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo ${GINKGO_OPTS}  integration

generate:
	@echo "# Generating counterfeits"
	@COUNTERFEITER_NO_GENERATE_WARNING=true go generate ./...

get-fmt-deps:
	go get golang.org/x/tools/cmd/goimports

importfmt: get-fmt-deps
	@echo "# Formatting the imports"
	@goimports -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`go fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

buildtools-force:
	@echo "# Installing build tools"
	go mod download
	go install github.com/square/certstrap
	go install github.com/onsi/ginkgo/v2/ginkgo
	go install github.com/maxbrunsfeld/counterfeiter/v6
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: buildtools
buildtools: golangci-lint
	@echo "# Installing build tools"
	@go mod download
	@which certstrap >/dev/null || go install github.com/square/certstrap
	@which ginkgo >/dev/null || go install github.com/onsi/ginkgo/v2/ginkgo
	@which counterfeiter >/dev/null || go install github.com/maxbrunsfeld/counterfeiter/v6

golangci-lint:
	@which golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: golangci-lint
	@golangci-lint run

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean
	@rm -rf build
