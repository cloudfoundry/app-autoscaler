SHELL := /bin/bash
GO := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD := GO111MODULE=off go
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')

CGO_ENABLED = 0
BUILDTAGS :=

build-%:
	CGO_ENABLED=$(CGO_ENABLED) $(GO_NOMOD) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go

build: build-scalingengine build-metricscollector build-metricsforwarder build-eventgenerator build-api build-metricsgateway build-metricsserver build-operator

check: fmt lint build test

test:
	ginkgo -r -race -randomizeAllSpecs

generate:
	COUNTERFEITER_NO_GENERATE_WARNING=true $(GO_NOMOD) generate ./...

get-fmt-deps: ## Install goimports
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

importfmt: get-fmt-deps
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true
