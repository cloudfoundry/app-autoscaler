SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS := -s
aes_terminal_font_yellow := \e[38;2;255;255;0m
aes_terminal_reset := \e[0m
VERSION ?= 0.0.0-rc.1
DEST ?= /tmp/build
MTAR_FILENAME ?= app-autoscaler-release-v$(VERSION).mtar
AUTOSCALER_DIR ?= $(shell pwd)/../..
CI_DIR ?= ${AUTOSCALER_DIR}/ci

GO_VERSION = $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_MINOR_VERSION = $(shell echo $(GO_VERSION) | cut --delimiter=. --field=2)
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')


MODULES ?= dbtasks,apiserver,eventgenerator,metricsforwarder,operator,scheduler,scalingengine

db_type ?= postgres
DB_HOST ?= localhost
DBURL := $(shell case "${db_type}" in\
			 (postgres) printf "postgres://postgres:postgres@${DB_HOST}/autoscaler?sslmode=disable"; ;; \
				 (mysql) printf "root@tcp(${DB_HOST})/autoscaler?tls=false"; ;; esac)

MAKEFILE_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
EXTENSION_FILE := $(shell mktemp)

export GOWORK=off
BUILDFLAGS := -ldflags '-linkmode=external'
export GOFIPS140=v1.0.0

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq | grep -Ev "vendor|integration")
test_dirs=$(shell find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
export GO111MODULE=on

.PHONY: clean-dbtasks package-dbtasks vendor-changelogs clean-scheduler package-scheduler clean mta-deploy mta-undeploy mta-build mta-logs

GINKGO_OPTS = -r --race --require-suite --randomize-all --cover ${OPTS}

# ogen generated OpenAPI clients and servers
openapi-generated-clients-and-servers-api-dir := ./api/apis/scalinghistory
openapi-generated-clients-and-servers-scalingengine-dir := ./scalingengine/apis/scalinghistory

openapi-spec-path := ../../api
openapi-specs-list = $(wildcard ${openapi-spec-path}/*.yaml)

openapi-generated-clients-and-servers-api-files = $(wildcard ${openapi-generated-clients-and-servers-api-dir}/*.go)
openapi-generated-clients-and-servers-scalingengine-files = $(wildcard ${openapi-generated-clients-and-servers-scalingengine-dir}/*.go)

.PHONY: generate-openapi-generated-clients-and-servers
generate-openapi-generated-clients-and-servers: ${openapi-generated-clients-and-servers-api-dir} ${openapi-generated-clients-and-servers-api-files} ${openapi-generated-clients-and-servers-scalingengine-dir} ${openapi-generated-clients-and-servers-scalingengine-files}
${openapi-generated-clients-and-servers-api-dir} ${openapi-generated-clients-and-servers-api-files} ${openapi-generated-clients-and-servers-scalingengine-dir} ${openapi-generated-clients-and-servers-scalingengine-files} &: $(wildcard ./scalingengine/apis/generate.go) $(wildcard ./api/apis/generate.go) ${openapi-specs-list} ./go.mod ./go.sum
	@echo "# Generating OpenAPI clients and servers"
	# $(wildcard ./api/apis/generate.go) causes the target to always being executed, no matter if file exists or not.
	# so let's don't fail if file can't be found, e.g. the eventgenerator bosh package does not contain it.
	go generate ./api/apis/generate.go || true
	go generate ./scalingengine/apis/generate.go || true

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

.PHONY: go-mod-vendor-mta
go-mod-vendor-mta: generate-openapi-generated-clients-and-servers
	GOFLAGS="-tags=!test" go mod vendor -e


# CGO_ENABLED := 1 is required to enforce dynamic linking which is a requirement of dynatrace.
build-%: generate-openapi-generated-clients-and-servers
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

.PHONY: generate-fakes
test: generate-fakes
	@echo "Running tests"
	APP_AUTOSCALER_TEST_RUN='true' go run github.com/onsi/ginkgo/v2/ginkgo -p ${GINKGO_OPTS} ${TEST} --skip-package='integration'

.PHONY: testsuite
testsuite: build-gorouterproxy
	@echo " - using DBURL=${DBURL} TEST=${TEST}"
	APP_AUTOSCALER_TEST_RUN='true' go run github.com/onsi/ginkgo/v2/ginkgo -p ${GINKGO_OPTS} ${TEST}

.PHONY: build-gorouterproxy
build-gorouterproxy:
	@echo "# building gorouterproxy"
	@CGO_ENABLED=1 go build $(BUILDTAGS) $(BUILDFLAGS) -o build/gorouterproxy integration/gorouterproxy/main.go

.PHONY: integration
integration: generate-fakes
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN='true' go run github.com/onsi/ginkgo/v2/ginkgo ${GINKGO_OPTS} integration

importfmt:
	@echo "# Formatting the imports"
	@go run golang.org/x/tools/cmd/goimports@latest -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`go fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

# This target depends on the fakes, because the tests are linted as well.
lint: generate-fakes
	readonly GOVERSION='${GO_VERSION}' ;\
	export GOVERSION ;\
	echo "Linting with Golang $${GOVERSION}" ;\
	golangci-lint run --config='../../.golangci.yaml' ${OPTS}

clean-dbtasks:
	pushd dbtasks; mvn clean; popd

package-dbtasks:
	pushd dbtasks; mvn package ; popd

clean-scheduler:
	pushd scheduler; mvn clean; popd

build-scheduler:
	pushd scheduler; mvn package -Dmaven.test.skip=true; popd

vendor-changelogs:
	cp $(MAKEFILE_DIR)/api/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/eventgenerator/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/operator/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/scalingengine/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/scheduler/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.

clean:
	@echo "# cleaning autoscaler"
	@go clean -cache -testcache
	@rm --force --recursive 'build'
	@rm --force --recursive 'fakes'
	@rm --force --recursive 'vendor'
	@rm --force --recursive "${openapi-generated-clients-and-servers-api-dir}"
	@rm --force --recursive "${openapi-generated-clients-and-servers-scalingengine-dir}"

mta-deploy: mta-build build-extension-file
	$(MAKE) -f metricsforwarder/Makefile set-security-group
	@echo "Deploying with extension file: $(EXTENSION_FILE)"
	@cf deploy $(DEST)/$(MTAR_FILENAME) --version-rule ALL -f --delete-services -e $(EXTENSION_FILE) -m $(MODULES)

mta-undeploy:
	@cf undeploy com.github.cloudfoundry.app-autoscaler-release -f

build-extension-file:
	echo "extension file at: $(EXTENSION_FILE)"
	$(MAKEFILE_DIR)/build-extension-file.sh $(EXTENSION_FILE);

mta-logs:
	rm -rf mta-*
	cf dmol --mta com.github.cloudfoundry.app-autoscaler-release --last 1
	vim mta-*

mta-build: mta-build-clean
	@echo "building mtar file for version: $(VERSION)"
	cp mta.tpl.yaml mta.yaml
	sed --in-place 's/MTA_VERSION/$(VERSION)/g' mta.yaml
	sed --in-place 's/GO_MINOR_VERSION/$(GO_MINOR_VERSION)/g' mta.yaml
	mkdir -p $(DEST)
	mbt build -t /tmp --mtar $(MTAR_FILENAME)
	@mv /tmp/$(MTAR_FILENAME) $(DEST)/$(MTAR_FILENAME)
	@echo '⚠️ The mta build is done. The mtar file is available at: $(DEST)/$(MTAR_FILENAME)'
	du -h $(DEST)/$(MTAR_FILENAME)

mta-build-clean:
	rm -rf mta_archives

.PHONY: cf-login
cf-login:
	@echo '⚠️ Please note that this login only works for cf and concourse,' \
		  'in spite of performing a login as well on bosh and credhub.' \
		  'The necessary changes to the environment get lost when make exits its process.'
	@${CI_DIR}/autoscaler/scripts/os-infrastructure-login.sh

deploy-apps: cf-login mta-deploy
