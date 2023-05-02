SHELL := /bin/bash
.SHELLFLAGS = -euo pipefail -c
MAKEFLAGS = -s
GO_VERSION := $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')
PACKAGE_DIRS := $(shell go list ./... | grep -v /vendor/ | grep -v e2e)
CGO_ENABLED = 1 # This is set to enforce dynamic linking which is a requirement of dynatrace.
BUILDTAGS :=
BUILDFLAGS := -ldflags '-linkmode=external'

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
test_dirs=$(shell   find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
export GO111MODULE=on

#TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/564 allow the tests to be run in parallel
#GINKGO_OPTS=-r --race --require-suite -p --randomize-all --cover

GINKGO_OPTS=-r --race --require-suite --randomize-all --cover ${OPTS}

build-%:
	@echo "# building $*"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go


build: $(addprefix build-,$(binaries))

build_tests: $(addprefix build_test-,$(test_dirs))

build_test-%:
	@echo " - building '$*' tests"
	@export build_folder=${PWD}/build/tests/$* &&\
	 mkdir -p $${build_folder} &&\
	 cd $* &&\
	 for package in $$(  go list ./... | sed 's|.*/autoscaler/$*|.|' | awk '{ print length, $$0 }' | sort -n -r | cut -d" " -f2- );\
	 do\
	   export test_file=$${build_folder}/$${package}.test;\
	   echo "   - compiling $${package} to $${test_file}";\
	   go test -c -o $${test_file} $${package};\
	 done;

check: fmt lint build test

ginkgo_check:
	@ current_version=$(shell ginkgo version | cut -d " " -f 3 | sed -E 's/([0-9]+\.[0-9]+)\..*/\1/');\
	expected_version=$(shell grep "ginkgo"  "../../.tool-versions" | cut -d " " -f 2 | sed -E 's/([0-9]+\.[0-9]+)\..*/\1/');\
	if [ "$${current_version}" != "$${expected_version}" ]; then \
        echo "WARNING: Expected to have ginkgo version '$${expected_version}.x' but we have $(shell ginkgo version)";\
    fi

test: ginkgo_check
	@echo "Running tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo -p ${GINKGO_OPTS} --skip-package=integration

testsuite: ginkgo_check
	APP_AUTOSCALER_TEST_RUN=true ginkgo -p ${GINKGO_OPTS} ${TEST}

.PHONY: integration
integration: ginkgo_check
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN=true ginkgo ${GINKGO_OPTS} integration

.PHONY: fakes generate
fakes: generate
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
	@which certstrap >/dev/null || go install github.com/square/certstrap
	@which ginkgo >/dev/null  && [ "v$(shell ginkgo version | awk '{ print $$3}')" = "$(shell cat go.mod | grep ginkgo | awk '{ print $$2}')" ] ||  go install github.com/onsi/ginkgo/v2/ginkgo
	@which counterfeiter >/dev/null || go install github.com/maxbrunsfeld/counterfeiter/v6

lint:
	@cd ../../; make lint_autoscaler OPTS=${OPTS}

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm -rf build
