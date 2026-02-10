SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS := -s
aes_terminal_font_yellow := \033[38;2;255;255;0m
aes_terminal_reset := \033[0m
VERSION ?= 0.0.0-rc.1
DEST ?= /tmp/build
TARGET_DIR ?= ./build
MTAR_FILENAME ?= app-autoscaler-release-v$(VERSION).mtar
ACCEPTANCE_TESTS_FILE ?= ${DEST}/app-autoscaler-acceptance-tests-v$(VERSION).tgz
CI ?= false

DEBUG ?= false
MYSQL_TAG := 8
POSTGRES_TAG := 16
GO_VERSION = $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')
MVN_OPTS ?= -Dmaven.test.skip=true

.PHONY: db.java-libs
db.java-libs:
	@echo 'Fetching db.java-libs'
	cd dbtasks && mvn -B --quiet package ${MVN_OPTS}

.PHONY: test-certs
test-certs: target/autoscaler_test_certs scheduler/src/test/resources/certs

.PHONY: build_all build_programs build_tests
build_all: build_programs build_tests
build_programs: build db.java-libs scheduler.build build-test-app

MODULES ?= dbtasks,apiserver,eventgenerator,metricsforwarder,operator,scheduler,scalingengine,acceptance-tests

db_type ?= postgres
DB_HOST ?= localhost
DBURL := $(shell case "${db_type}" in\
			 (postgres) printf "postgres://postgres:postgres@${DB_HOST}/autoscaler?sslmode=disable"; ;; \
				 (mysql) printf "root@tcp(${DB_HOST})/autoscaler?tls=false"; ;; esac)

MAKEFILE_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

export GOWORK=off
BUILDFLAGS := -ldflags '-linkmode=external'
export GOFIPS140=v1.0.0

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq | grep -Ev "vendor|integration|acceptance|test-app")
test_dirs=$(shell find . -name "*_test.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq)
export GO111MODULE=on

.PHONY: dbtasks.clean package-dbtasks vendor-changelogs scheduler.clean package-scheduler clean mta-deploy mta-undeploy mta-build mta-logs

GINKGO_OPTS = -r --race --require-suite --randomize-all ${OPTS}

# ogen generated OpenAPI clients and servers
openapi-generated-clients-and-servers-api-dir := ./api/apis/scalinghistory
openapi-generated-clients-and-servers-scalingengine-dir := ./scalingengine/apis/scalinghistory

openapi-spec-path := ../../api
openapi-specs-list = $(wildcard ${openapi-spec-path}/*.yaml)

openapi-generated-clients-and-servers-api-files = $(wildcard ${openapi-generated-clients-and-servers-api-dir}/*.go)
openapi-generated-clients-and-servers-scalingengine-files = $(wildcard ${openapi-generated-clients-and-servers-scalingengine-dir}/*.go)

.PHONY:
scheduler.build:
	@make --directory=scheduler build

.PHONY: check-type
check-db_type:
	@case "${db_type}" in\
	 (mysql|postgres) echo " - using db_type:${db_type}"; ;;\
	 (*) echo "ERROR: db_type needs to be one of mysql|postgres"; exit 1;;\
	 esac

.PHONY: generate-openapi-generated-clients-and-servers
generate-openapi-generated-clients-and-servers: ${openapi-generated-clients-and-servers-api-dir} ${openapi-generated-clients-and-servers-api-files} ${openapi-generated-clients-and-servers-scalingengine-dir} ${openapi-generated-clients-and-servers-scalingengine-files}
${openapi-generated-clients-and-servers-api-dir} ${openapi-generated-clients-and-servers-api-files} ${openapi-generated-clients-and-servers-scalingengine-dir} ${openapi-generated-clients-and-servers-scalingengine-files} &: $(wildcard ./scalingengine/apis/generate.go) $(wildcard ./api/apis/generate.go) ${openapi-specs-list} ./go.mod ./go.sum
	@echo "# Generating OpenAPI clients and servers"
	# $(wildcard ./api/apis/generate.go) causes the target to always being executed, no matter if file exists or not.
	# so let's don't fail if file can't be found, e.g. the eventgenerator bosh package does not contain it.
	go generate ./api/apis/generate.go || true
	go generate ./scalingengine/apis/generate.go || true


.PHONY: generate-fakes
generate-fakes: autoscaler.generate-fakes test-app.generate-fakes

# The presence of the subsequent directory indicates whether the fakes still need to be generated
# or not.
app-fakes-dir := ./fakes
app-fakes-files = $(wildcard ${app-fakes-dir}/*.go)
fake-relevant-go-files = $(shell find . -type f -name '*.go' \
	! -path './acceptance/*' \
	! -path './fakes/*' \
	! -path './integration/*' \
	! -path './target/*' \
	! -path './test-certs/*' \
	! -path './testhelpers/*' \
	! -path './vendor/*' \
	! -name '*_test.go')
.PHONY: autoscaler.generate-fakes
autoscaler.generate-fakes: ${app-fakes-dir} ${app-fakes-files}
${app-fakes-dir} ${app-fakes-files} &: ./go.mod ./go.sum ${fake-relevant-go-files}
	@echo '# Generating counterfeits'
	mkdir -p '${app-fakes-dir}'
	COUNTERFEITER_NO_GENERATE_WARNING='true' GOFLAGS='-mod=mod' go generate './...'
	@touch '${app-fakes-dir}' # Ensure that the folder-modification-timestamp gets updated.

.PHONY: test-app.generate-fakes
test-app.generate-fakes:
	make --directory='acceptance/assets/app/go_app' generate-fakes

go_deps_without_generated_sources = $(shell find . -type f -name '*.go' \
																| grep --invert-match --extended-regexp \
																		--regexp='${app-fakes-dir}|${openapi-generated-clients-and-servers-api-dir}|${openapi-generated-clients-and-servers-scalingengine-dir}')


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
go-mod-tidy: ./go.mod ./go.sum ${go_deps_without_generated_sources} acceptance.go-mod-tidy test-app.go-mod-tidy
	@echo -ne '${aes_terminal_font_yellow}' \
		'⚠️ Warning: The client-fakes generated from the openapi-specification may be\n' \
		'outdated. Please consider re-generating them, if this is relevant.' \
		'${aes_terminal_reset}'
	go mod tidy

.PHONY: acceptance.go-mod-tidy
acceptance.go-mod-tidy:
	make --directory='acceptance' go-mod-tidy

.PHONY: test-app.go-mod-tidy
test-app.go-mod-tidy:
	make --directory='acceptance/assets/app/go_app' go-mod-tidy

go-vendoring-folder := ./vendor
go-vendored-files = $(shell find '${go-vendoring-folder}' -type f -name '*.go' 2> '/dev/null')
## This does not work: go-vendored-files = $(wildcard ${go-vendoring-folder}/**/*.go)

.PHONY: go-mod-vendor
go-mod-vendor: generate-fakes
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

test: autoscaler.test scheduler.test test-acceptance-unit ## Run all unit tests

autoscaler.test: check-db_type init-db test-certs generate-fakes build-gorouterproxy
	@echo ' - using DBURL=${DBURL} TEST=${TEST}'
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' ginkgo run -p ${GINKGO_OPTS} --skip-package='integration,acceptance' ${TEST}

test-autoscaler-suite: check-db_type init-db test-certs build-gorouterproxy
	@echo " - using DBURL=${DBURL} TEST=${TEST}"
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' ginkgo run -p ${GINKGO_OPTS} ${TEST}

test-acceptance-unit:
	@make --directory=acceptance test-unit

gorouter-proxy.program := ./build/gorouterproxy
gorouter-proxy.source := integration/gorouterproxy/main.go
.PHONY: build-gorouterproxy
build-gorouterproxy: ${gorouter-proxy.program}
${gorouter-proxy.program}: ./go.mod ./go.sum ${gorouter-proxy.source}
	@echo "# building gorouterproxy"
	@CGO_ENABLED=1 go build $(BUILDTAGS) $(BUILDFLAGS) -o '${gorouter-proxy.program}' '${gorouter-proxy.source}'

.PHONY: integration
integration: generate-fakes init-db test-certs build_all build-gorouterproxy
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' ginkgo ${GINKGO_OPTS} integration DBURL="${DBURL}"

.PHONY: init-db
init-db: check-db_type start-db db.java-libs target/init-db-${db_type}
target/init-db-${db_type}:
	@./scripts/initialise_db.sh '${db_type}'
	@mkdir -p target
	@touch $@

.PHONY: provision-db
provision-db: ## Provision a database on a remote Postgres server (requires POSTGRES_IP)
	@./scripts/provision_db.sh

.PHONY: deprovision-db
deprovision-db: ## Deprovision a database on a remote Postgres server (requires POSTGRES_IP)
	@./scripts/deprovision_db.sh

importfmt:
	@echo "# Formatting the imports"
	@go run golang.org/x/tools/cmd/goimports@latest -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`go fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

# This target depends on the fakes, because the tests are linted as well.

package-dbtasks:
	pushd dbtasks; mvn -B --quiet package ${MVN_OPTS}; popd

build-scheduler:
	pushd scheduler; mvn -B --quiet package -Dmaven.test.skip=true; popd

vendor-changelogs:
	cp $(MAKEFILE_DIR)/api/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/eventgenerator/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/operator/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/scalingengine/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.
	cp $(MAKEFILE_DIR)/scheduler/db/* $(MAKEFILE_DIR)/dbtasks/src/main/resources/.

clean: dbtasks.clean scheduler.clean
	@echo "# cleaning autoscaler"
	@rm --force --recursive "${openapi-generated-clients-and-servers-api-dir}"
	@rm --force --recursive "${openapi-generated-clients-and-servers-scalingengine-dir}"
	@go clean -cache -testcache
	@rm --force --recursive 'fakes'
	@rm --force --recursive 'test-certs'
	@rm --force --recursive 'target'
	@rm --force --recursive 'vendor'

dbtasks.clean:
	pushd dbtasks; mvn -B clean; popd

scheduler.clean:
	pushd scheduler; mvn -B --quiet clean; popd

schema-files := $(shell find ./api/policyvalidator -type f -name '*.json')
flattened-schema-file := ${DEST}/bind-request.schema.json
BIND_REQ_SCHEMA_VERSION ?= v0.1
bind-request-schema: ${flattened-schema-file}
${flattened-schema-file}: ${schema-files}
	mkdir -p "$$(dirname ${flattened-schema-file})"
	flatten_json-schema './api/policyvalidator/json-schema/${BIND_REQ_SCHEMA_VERSION}/meta.schema.json' \
	> '${flattened-schema-file}'
	echo '🔨 File created: ${flattened-schema-file}'

mta-deploy:
	$(MAKEFILE_DIR)/scripts/mta-deploy.sh

set-security-group:
	$(MAKEFILE_DIR)/scripts/set-security-group.sh

mta-undeploy:
	@cf undeploy com.github.cloudfoundry.app-autoscaler-release -f

build-extension-file:
	$(MAKEFILE_DIR)/scripts/build-extension-file.sh

mta-logs:
	rm -rf mta-*
	cf dmol --mta com.github.cloudfoundry.app-autoscaler-release --last 1
	vim mta-*

mta-build: mta-build-clean
	@$(MAKEFILE_DIR)/scripts/mta-build.sh

mta-build-clean:
	rm -rf mta_archives

.PHONY: clean-build
clean-build: ## Clean the build directory
	@echo ' - cleaning build directory'
	@rm -rf build

.PHONY: release-draft
release-draft: ## Create a draft GitHub release without artifacts
		./scripts/release.sh

.PHONY: create-assets
create-assets: ## Create release assets (mtar and acceptance tests), please provide `VERSION` as environment-variable.
		./scripts/create-assets.sh

.PHONY: release-promote
release-promote: create-assets ## Promote draft release to final and upload assets
		PROMOTE_DRAFT=true ./scripts/release.sh

.PHONY: acceptance-release
acceptance-release: generate-fakes clean-acceptance go-mod-tidy go-mod-vendor build-test-app
	@echo " - building acceptance test release '${VERSION}' to dir: '${DEST}' "
	@mkdir -p ${DEST}
	# Build for linux_amd64 by default (CF tasks platform)
	@export TARGET_OS=$${TARGET_OS:-linux} TARGET_ARCH=$${TARGET_ARCH:-amd64}; \
	echo " - Building for OS: $$TARGET_OS, ARCH: $$TARGET_ARCH"; \
	./scripts/compile-acceptance-tests.sh
	# Create scripts directory and copy wrapper script
	@mkdir -p build/acceptance/scripts build/acceptance/bin
	@cp scripts/run-acceptance-tests-task.sh build/acceptance/scripts/
	@chmod +x build/acceptance/scripts/run-acceptance-tests-task.sh
	# Download and bundle CF CLI for linux_amd64
	@echo "Downloading CF CLI for linux_amd64..."
	@curl -L "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=v8" -o /tmp/cf-cli.tgz
	@tar -xzf /tmp/cf-cli.tgz -C build/acceptance/bin/
	@chmod +x build/acceptance/bin/cf
	@rm /tmp/cf-cli.tgz
	# Create tarball
	@tar --create --auto-compress --file="${ACCEPTANCE_TESTS_FILE}" -C build acceptance


.PHONY: mta-release
mta-release: generate-fakes mta-build
	@echo " - building mtar release '${VERSION}' to dir: '${DEST}' "

clean-acceptance:
	@echo ' - cleaning acceptance (⚠️ This keeps the file “acceptance/acceptance_config.json” if present!)'
	@rm acceptance/ginkgo* &> /dev/null || true
	@rm -rf acceptance/results &> /dev/null || true

.PHONY: cf-admin-login
cf-admin-login:
	@echo '⚠️ Please note that this login only works for cf and concourse,' \
		  'in spite of performing a login as well on bosh and credhub.' \
		  'The necessary changes to the environment get lost when make exits its process.'
	@${MAKEFILE_DIR}/scripts/os-infrastructure-login.sh

.PHONY: cf-org-manager-login
cf-org-manager-login:
	@echo '⚠️ Please note that this login only works for cf and concourse,' \
		  'in spite of performing a login as well on bosh and credhub.' \
		  'The necessary changes to the environment get lost when make exits its process.'
	@${MAKEFILE_DIR}/scripts/org-manager-login.sh


.PHONY: start-db
start-db: check-db_type target/start-db-${db_type}_CI_${CI} waitfor_${db_type}_CI_${CI}
	@echo " SUCCESS"


.PHONY: waitfor_postgres_CI_false waitfor_postgres_CI_true
target/start-db-postgres_CI_false:
	if [ ! "$(shell docker ps -q -f name="^${db_type}")" ] ;\
	then \
		if [ "$(shell docker ps -aq -f status=exited -f name="^${db_type}")" ]; \
		then \
			docker rm ${db_type}; \
		fi;\
		echo " - starting docker for ${db_type}";\
		docker run -p 5432:5432 --name postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_DB=autoscaler \
			--health-cmd pg_isready \
			--health-interval 1s \
			--health-timeout 2s \
			--health-retries 10 \
			-d \
			postgres:${POSTGRES_TAG} \
			-c 'max_connections=1000' >/dev/null;\
	else \
		echo " - $@ already up'";\
	fi;
	@mkdir -p target
	@touch $@

target/start-db-postgres_CI_true:
	@echo " - $@ already up'"

waitfor_postgres_CI_false:
	@echo -n " - waiting for ${db_type} ."
	@COUNTER=0; until $$(docker exec postgres pg_isready &>/dev/null) || [ $$COUNTER -gt 10 ]; do echo -n "."; sleep 1; let COUNTER+=1; done;\
	if [ $$COUNTER -gt 10 ]; then echo; echo "Error: timed out waiting for postgres. Try \"make clean\" first." >&2 ; exit 1; fi

waitfor_postgres_CI_true:
	@echo " - no ci postgres checks"

.PHONY: waitfor_mysql_CI_false waitfor_mysql_CI_true
target/start-db-mysql_CI_false:
	@if [  ! "$(shell docker ps -q -f name="^${db_type}")" ]; then \
		if [ "$(shell docker ps -aq -f status=exited -f name="^${db_type}")" ]; then \
			docker rm ${db_type}; \
		fi;\
		echo " - starting docker for ${db_type}";\
		docker pull mysql:${MYSQL_TAG}; \
		docker run -p 3306:3306  --name mysql \
			-e MYSQL_ALLOW_EMPTY_PASSWORD=true \
			-e MYSQL_DATABASE=autoscaler \
			-d \
			mysql:${MYSQL_TAG} \
			>/dev/null;\
	else echo " - $@ already up"; fi;
	@mkdir -p target
	@touch $@
target/start-db-mysql_CI_true:
	@echo " - $@ already up'"
waitfor_mysql_CI_false:
	@echo -n " - waiting for ${db_type} ."
	@until docker exec mysql mysqladmin ping &>/dev/null ; do echo -n "."; sleep 1; done
	@echo " SUCCESS"
	@echo -n " - Waiting for table creation ."
	@until [[ ! -z `docker exec mysql mysql -qfsBe "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='autoscaler'" 2> /dev/null` ]]; do echo -n "."; sleep 1; done
waitfor_mysql_CI_true:
	@echo -n " - Waiting for table creation (DB_HOST=${DB_HOST})"
	@which mysql > /dev/null || { echo "ERROR: mysql client not found"; exit 1; }
	@T=0;\
	until mysql -u "root" -h "${DB_HOST}" --port=3306 -qfsBe "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='autoscaler'" 2>&1 | grep -q autoscaler \
		|| [[ $${T} -gt 120 ]];\
	do \
		echo -n "."; \
		if [[ $$((T % 10)) -eq 0 ]] && [[ $${T} -gt 0 ]]; then \
			echo -n " [$$T/120s, trying: mysql -u root -h ${DB_HOST} --port=3306] "; \
		fi; \
		sleep 1; \
		T=$$((T+1)); \
	done; \
	if [[ $${T} -gt 120 ]]; then \
		echo ""; \
		echo "ERROR: Mysql timed out creating database after 120 seconds"; \
		echo "Attempted connection: mysql -u root -h ${DB_HOST} --port=3306"; \
		echo "Trying one more time with verbose output:"; \
		mysql -u "root" -h "${DB_HOST}" --port=3306 -vvv -e "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='autoscaler'" 2>&1 || true; \
		exit 1; \
	fi

.PHONY: stop-db
stop-db: check-db_type
	@echo " - Stopping ${db_type}"
	@rm target/start-db-${db_type} &> /dev/null || echo " - Seems the make target was deleted stopping anyway!"
	@docker rm -f ${db_type} &> /dev/null || echo " - we could not stop and remove docker named '${db_type}'"

target/autoscaler_test_certs:
	@mkdir -p target
	@./scripts/generate_test_certs.sh
	@touch $@

scheduler/src/test/resources/certs:
	@./scheduler/scripts/generate_unit_test_certs.sh

.PHONY: build-test-app
build-test-app:
	@make --directory=acceptance/assets/app/go_app build

.PHONY: acceptance.build_tests
acceptance.build_tests:
	@make --directory=acceptance build_tests

.PHONY: acceptance.tests-cleanup
acceptance.tests-cleanup:
	@make --directory=acceptance acceptance-tests-cleanup

# This target is defined here rather than directly in the component “scheduler” itself, because it depends on targets outside that component. In the future, it will be moved back to that component and reference a dependency to a Makefile on the same level – the one for the component it depends on.
.PHONY: scheduler.test
scheduler.test: check-db_type scheduler.test-certificates init-db
	@make --directory=scheduler test

.PHONY: scheduler.test-certificates
scheduler.test-certificates:
	make --directory=scheduler test-certificates

lint: lint-go lint-actions lint-markdown
.PHONY: lint-go
lint-go: generate-fakes acceptance.lint test-app.lint gorouterproxy.lint
	readonly GOVERSION='${GO_VERSION}' ;\
	export GOVERSION ;\
	echo "Linting with Golang $${GOVERSION}" ;\
	golangci-lint run --config='.golangci.yaml' ${OPTS}

.PHONY: lint-actions
lint-actions:
	@echo " - linting GitHub actions"
	actionlint

acceptance.lint:
	@echo 'Linting acceptance-tests …'
	make --directory='acceptance' lint

test-app.lint:
	@echo 'Linting test-app …'
	make --directory='acceptance/assets/app/go_app' lint

.PHONY: build-acceptance-tests
build-acceptance-tests:
	@make --directory='acceptance' build_tests

.PHONY: acceptance-tests
acceptance-tests: build-test-app acceptance-tests-config ## Run acceptance tests against OSS dev environment (requrires a previous deployment of the autoscaler)
	@make --directory='acceptance' run-acceptance-tests

.PHONY: acceptance-cleanup
acceptance-cleanup:
	@make --directory='acceptance' acceptance-tests-cleanup

.PHONY: acceptance-tests-config
acceptance-tests-config:
	make --directory='acceptance' acceptance-tests-config

.PHONY: mta-acceptance-tests
mta-acceptance-tests: ## Run MTA acceptance tests in parallel via CF tasks
	@$(MAKEFILE_DIR)/scripts/run-mta-acceptance-tests.sh

.PHONY: setup-org-manager-user
setup-org-manager-user: ## Setup org manager user with OrgManager and SpaceDeveloper roles (password from CredHub)
	DEBUG="${DEBUG}" ./scripts/setup-org-manager-user.sh

# 🚧 To-do: These targets don't exist here!
.PHONY: deploy-autoscaler deploy-autoscaler-bosh

.PHONY: register-broker
register-broker:
	DEBUG="${DEBUG}" ./scripts/register-broker.sh

.PHONY: deploy-cleanup
deploy-cleanup:
	DEBUG="${DEBUG}" ./scripts/cleanup-autoscaler.sh

.PHONY: deploy-apps
deploy-apps:
	DEBUG="${DEBUG}" ./scripts/deploy-apps.sh

.PHONY: lint-markdown
lint-markdown:
	@echo " - linting markdown files"
	@markdownlint-cli2 .

gorouterproxy.lint:
	@echo " - linting: gorouterproxy"
	@pushd integration/gorouterproxy >/dev/null && golangci-lint run --config='${lint_config}' $(OPTS)

validate-openapi-specs: $(wildcard ./openapi/*.openapi.yaml)
	for file in $^ ; do \
		redocly lint --extends=minimal --format=$(if $(GITHUB_ACTIONS),github-actions,codeframe) "$${file}" ; \
	done
