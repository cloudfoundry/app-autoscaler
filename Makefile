SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS := -s
aes_terminal_font_yellow := \e[38;2;255;255;0m
aes_terminal_reset := \e[0m

GO_VERSION = $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')

DB_HOST ?= localhost
DBURL ?= "postgres://postgres:postgres@${DB_HOST}/autoscaler?sslmode=disable"

export GOWORK=off
BUILDFLAGS := -ldflags '-linkmode=external'

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq | grep -v vendor)
test_dirs=$(shell find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
export GO111MODULE=on

#TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/564 allow the tests to be run in parallel
#GINKGO_OPTS=-r --race --require-suite -p --randomize-all --cover

GINKGO_OPTS = -r --race --require-suite --randomize-all --cover ${OPTS}
GINKGO_VERSION = v$(shell cat ../../.tool-versions | grep ginkgo  | cut --delimiter=' ' --fields='2')


# ogen generated OpenAPI clients and servers
openapi-generated-clients-and-servers-dir := ./helpers/apis/scalinghistory
openapi-spec-path := ../../api
openapi-specs-list = $(wildcard ${openapi-spec-path}/*.yaml)

openapi-generated-clients-and-servers-files = $(wildcard ${openapi-generated-clients-and-servers-dir}/*.go)

.PHONY: generate-openapi-generated-clients-and-servers
generate-openapi-generated-clients-and-servers: ${openapi-generated-clients-and-servers-dir} ${openapi-generated-clients-and-servers-files}
${openapi-generated-clients-and-servers-dir} ${openapi-generated-clients-and-servers-files} &: $(wildcard ./helpers/apis/generate.go) ${openapi-specs-list} ./go.mod ./go.sum
	@echo "# Generating OpenAPI clients and servers"
	# $(wildcard ./helpers/apis/generate.go) causes the target to always being executed, no matter if file exists or not.
	# so let's don't fail if file can't be found, e.g. the eventgenerator bosh package does not contain it.
	go generate ./helpers/apis/generate.go || true

# The presence of the subsequent directory indicates whether the fakes still need to be generated
# or not.
app-fakes-dir := ./fakes
app-fakes-files = $(wildcard ${app-fakes-dir}/*.go)
.PHONY: generate-fakes
generate-fakes: ${app-fakes-dir} ${app-fakes-files} ${openapi-generated-clients-and-servers-dir}
${app-fakes-dir} ${app-fakes-files} &: ./go.mod ./go.sum ./generate-fakes.go
	@echo "# Generating counterfeits"
	mkdir -p '${app-fakes-dir}'
	COUNTERFEITER_NO_GENERATE_WARNING='true' go generate './...'


go_deps_without_generated_sources = $(shell find . -type f -name '*.go' \
																| grep --invert-match --extended-regexp \
																		--regexp='${app-fakes-dir}|${openapi-generated-clients-and-servers-dir}')

# This target should depend additionally on `${app-fakes-dir}` and on `${app-fakes-files}`. However
# this is not defined here. The reason is, that for `go-mod-tidy` the generated fakes need to be
# present but fortunately not necessarily up-to-date. This is fortunate because the generation of
# the fake requires the files `go.mod` and `go.sum` to be already tidied up, introducing a cyclic
# dependency otherwise. But that would make any modification to `go.mod` or `go.sum`
# impossible. This definition now makes it possible to update `go.mod` and `go.sum` as follows:
#  1. `make generate-fakes`
#  2. Update `go.mod` and/or `go.sum`
#  3. `make go-mod-tidy`
#  4. Optionally: `make generate-fakes` to update the fakes as well.
.PHONY: go-mod-tidy
go-mod-tidy: ./go.mod ./go.sum ${go_deps_without_generated_sources}
	@echo -ne '${aes_terminal_font_yellow}'
	@echo -e '⚠️ Warning: The client-fakes generated from the openapi-specification may be\n' \
					 'outdated. Please consider re-generating them, if this is relevant.'
	@echo -ne '${aes_terminal_reset}'
	go mod tidy



go-vendoring-folder := ./vendor
go-vendored-files = $(shell find '${go-vendoring-folder}' -type f -name '*.go' 2> '/dev/null')
## This does not work: go-vendored-files = $(wildcard ${go-vendoring-folder}/**/*.go)

.PHONY: go-mod-vendor
go-mod-vendor: ${go-vendoring-folder} ${go-vendored-files}
${go-vendoring-folder} ${go-vendored-files} &: ${app-fakes-dir} ${app-fakes-files}
	go mod vendor

build-cf-%:
	@echo "# building for cf $*"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $*/$* $*/cmd/$*/main.go

# CGO_ENABLED := 1 is required to enforce dynamic linking which is a requirement of dynatrace.
build-%: ${openapi-generated-clients-and-servers-dir} ${openapi-generated-clients-and-servers-files}
	@echo "# building $*"
	@CGO_ENABLED=1 go build $(BUILDTAGS) $(BUILDFLAGS) -o build/$* $*/cmd/$*/main.go


build: $(addprefix build-,$(binaries))

build_tests: $(addprefix build_test-,$(test_dirs))

build_test-%: generate-fakes
	@echo " - building '$*' tests"
	@export build_folder=${PWD}/build/tests/$* &&\
	 mkdir -p $${build_folder} &&\
	 cd $* &&\
	 for package in $$(go list ./... | sed 's|.*/autoscaler/$*|.|' | awk '{ print length, $$0 }' | sort -n -r | cut -d" " -f2- );\
	 do\
		 export test_file=$${build_folder}/$${package}.test;\
		 echo "   - compiling $${package} to $${test_file}";\
		 go test -c -o $${test_file} $${package};\
	 done;

check: fmt lint build test

test: generate-fakes
	@echo "Running tests"
	APP_AUTOSCALER_TEST_RUN='true' go run 'github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}' -p ${GINKGO_OPTS} --skip-package='integration'

testsuite: generate-fakes
	@echo " - using DBURL=${DBURL} TEST=${TEST}"
	APP_AUTOSCALER_TEST_RUN='true' go run 'github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}' -p ${GINKGO_OPTS} ${TEST}

.PHONY: integration
integration: generate-fakes
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN='true' go run 'github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}' ${GINKGO_OPTS} integration

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
	@rm --force --recursive 'vendor'
	@rm --force --recursive "${openapi-generated-clients-and-servers-dir}"

.PHONY: mta-deploy
mta-deploy: mta-build build
	@echo "Deploying mta"
	@echo " CF_TRACE=true cf deploy MTA mta_archives/*.mtar -e config.mtaext"
	CF_TRACE=true cf deploy mta_archives/com.github.cloudfoundry.app-autoscaler-release_0.0.1.mtar -e config.mtaext


.PHONY: mta-build
mta-build:
	mbt build
