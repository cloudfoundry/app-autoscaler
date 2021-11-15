SHELL := /bin/bash
GO := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD := GO111MODULE=off go
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/ | grep -v e2e)
PATH:=${HOME}/go/bin:${PATH}

CGO_ENABLED = 0
BUILDTAGS :=

build-%:
	@echo "# building $*"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go

build: build-scalingengine build-metricsforwarder build-eventgenerator build-api build-metricsgateway\
		build-metricsserver build-operator build-cred_helper

check: fmt lint build test

test:
	@echo "Running tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo -r -race -requireSuite -randomizeAllSpecs -cover --skipPackage=integration

testsuite:
	APP_AUTOSCALER_TEST_RUN=true ginkgo -r -race -randomizeAllSpecs $(TEST)

.PHONY: integration
integration:
	@echo "# Running integration tests"
	@APP_AUTOSCALER_TEST_RUN=true ginkgo -r -race -requireSuite -randomizeAllSpecs integration

generate:
	@echo "# Generating counterfeits"
	@COUNTERFEITER_NO_GENERATE_WARNING=true $(GO) generate ./...

get-fmt-deps: ## Install goimports
	@$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

importfmt: get-fmt-deps
	@echo "# Formatting the imports"
	@goimports -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

buildtools-force:
	@echo "# Installing build tools"
	$(GO) mod download
	$(GO) install github.com/square/certstrap
	$(GO) install github.com/onsi/ginkgo/ginkgo
	$(GO) install github.com/maxbrunsfeld/counterfeiter/v6
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: buildtools
buildtools:
	@echo "# Installing build tools"
	@$(GO) mod download
	@which certstrap >/dev/null || $(GO) install github.com/square/certstrap
	@which ginkgo >/dev/null || $(GO) install github.com/onsi/ginkgo/ginkgo
	@which counterfeiter >/dev/null || $(GO) install github.com/maxbrunsfeld/counterfeiter/v6
	@which golangci-lint >/dev/null || $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint:
	@golangci-lint run

lint-fix:
	@golangci-lint run --fix

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@${GO} clean
	@rm -rf build
