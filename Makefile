SHELL := /bin/bash
GO_VERSION := $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')
PACKAGE_DIRS := $(shell go list ./... | grep -v /vendor/ | grep -v e2e)
CGO_ENABLED = 0
BUILDTAGS :=
export GO111MODULE=on

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
		build-operator

build_tests:	build_test-scalingengine\
		build_test-metricsforwarder\
		build_test-eventgenerator\
		build_test-api\
		build_test-metricsgateway\
		build_test-metricsserver\
		build_test-operator\
		build_test-db

build_test-%:
	@echo " - $* tests"
	@export build_folder=${PWD}/build/$* && \
 	mkdir -p $${build_folder} && \
  	cd $* && \
  	for package in $$( find . -name "*.go" -exec dirname {} \; | sort | uniq  ); \
 	  do \
 	    name=tests_$$(echo "$${package}" | sed "s|\.|$*|" | sed 's|/|_|');\
 	    echo " -- compiling $*/$${package} to $${build_folder}/$${name}"; \
 	  	go test -c -o $${build_folder}/$${name} $${package};\
 	done;

check: fmt lint build test

ginkgo_check:
	@ current_version=$(shell ginkgo version | cut -d " " -f 3 | sed -E 's/([0-9]+\.[0-9]+)\..*/\1/');\
	expected_version=$(shell cat go.mod | grep "ginkgo"  | cut -d " " -f 2 | sed -E 's/v([0-9]+\.[0-9]+)\..*/\1/');\
	if [ "$${current_version}" != "$${expected_version}" ]; then \
        echo "ERROR: Expected to have ginkgo version '$${expected_version}.x' but we have $(shell ginkgo version)";\
        exit 1;\
    fi

test: ginkgo_check
	@echo "Running tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo -p ${GINKGO_OPTS} --skip-package=integration

testsuite: ginkgo_check
	APP_AUTOSCALER_TEST_RUN=true ginkgo -p ${GINKGO_OPTS} ${TEST}

.PHONY: integration
integration: ginkgo_check
	@echo "# Running integration tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo ${GINKGO_OPTS}  integration

generate:
	@echo "# Generating counterfeits"
	@COUNTERFEITER_NO_GENERATE_WARNING=true go generate ./...

importfmt:
	@echo "# Formatting the imports"
	@go run golang.org/x/tools/cmd/goimports@latest -w $(GO_DEPENDENCIES)

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
buildtools:
	@echo "# Installing build tools"
	@go mod download
	@which certstrap >/dev/null || go install github.com/square/certstrap
	@which ginkgo >/dev/null || go install github.com/onsi/ginkgo/v2/ginkgo
	@which counterfeiter >/dev/null || go install github.com/maxbrunsfeld/counterfeiter/v6

lint:
	@cd ../../; make lint_autoscaler OPTS=${OPTS}

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build

