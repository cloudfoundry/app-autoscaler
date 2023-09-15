SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS := -s
aes_terminal_font_yellow := \e[38;2;255;255;0m
aes_terminal_reset := \e[0m

GO_VERSION = $(shell go version | sed --expression='s/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')

# `CGO_ENABLED := 1` is required to enforce dynamic linking which is a requirement of dynatrace.
CGO_ENABLED := 1
BUILDTAGS :=
export GOWORK=off
BUILDFLAGS := -ldflags '-linkmode=external'

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq | grep -v vendor)
test_dirs=$(shell   find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
export GO111MODULE=on

#TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/564 allow the tests to be run in parallel
#GINKGO_OPTS=-r --race --require-suite -p --randomize-all --cover

GINKGO_OPTS=-r --race --require-suite --randomize-all --cover ${OPTS}
GINKGO_VERSION=v$(shell cat ../../.tool-versions | grep ginkgo  | cut -d " " -f 2 )


app-fakes-dir := ./fakes
app-fakes-files := $(wildcard ${app-fakes-dir}/*.go)

.PHONY: generate-fakes
generate-fakes: ${app-fakes-dir} ${app-fakes-files}
${app-fakes-dir} ${app-fakes-files} &: ./generate-fakes.go
	@echo -ne '${aes_terminal_font_yellow}'
	@echo -e '⚠️ The client-fakes generated from the openapi-specification depend on\n' \
					 'the files ./go.mod and ./go.sum. This has not been reflected in this\n' \
					 'make-target to avoid cyclic dependencies because `go mod tidy`, which\n' \
					 'modifies both files, depends itself on the client-fakes.'
	@echo -ne '${aes_terminal_reset}'
	@echo "# Generating counterfeits"
	mkdir -p '${app-fakes-dir}'
	COUNTERFEITER_NO_GENERATE_WARNING='true' go generate ./...



.PHONY: go-mod-tidy
go-mod-tidy: ${app-fakes-dir} ${app-fakes-files}
	go mod tidy



build-%:
	@echo "# building $*"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go


build: $(addprefix build-,$(binaries))

build_tests: $(addprefix build_test-,$(test_dirs))

build_test-%: generate
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

test: generate
	@echo "Running tests"
	@APP_AUTOSCALER_TEST_RUN=true go run github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION} -p ${GINKGO_OPTS} --skip-package=integration

testsuite: generate
	APP_AUTOSCALER_TEST_RUN=true go run github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION} -p ${GINKGO_OPTS} ${TEST}

.PHONY: integration
integration: generate
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN=true go run github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION} ${GINKGO_OPTS} integration

.PHONY: generate
generate:
	@echo "# Generating counterfeits"
	@COUNTERFEITER_NO_GENERATE_WARNING=true go generate ./...

importfmt:
	@echo "# Formatting the imports"
	@go run golang.org/x/tools/cmd/goimports@latest -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`go fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

lint:
	@cd ../../; make lint_autoscaler OPTS=${OPTS}

.PHONY: clean
clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm --force --recursive 'build'
	@rm --force --recursive 'fakes'
