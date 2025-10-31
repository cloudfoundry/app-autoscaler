SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS := -s
aes_terminal_font_yellow := \e[38;2;255;255;0m
aes_terminal_reset := \e[0m
VERSION ?= 0.0.0-rc.1
DEST ?= /tmp/build
MTAR_FILENAME ?= app-autoscaler-release-v$(VERSION).mtar
CI ?= false
CI_DIR ?= ${AUTOSCALER_DIR}/ci

DEBUG := false
MYSQL_TAG := 8
POSTGRES_TAG := 16
GO_VERSION = $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_MINOR_VERSION = $(shell echo $(GO_VERSION) | cut --delimiter=. --field=2)
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')
MVN_OPTS ?= -Dmaven.test.skip=true

.PHONY: db.java-libs
db.java-libs:
	@echo 'Fetching db.java-libs'
	cd dbtasks && mvn --quiet package ${MVN_OPTS}

.PHONY: test-certs
test-certs: target/autoscaler_test_certs scheduler/src/test/resources/certs

.PHONY: build_all build_programs build_tests
build_all: build_programs build_tests
build_programs: build db.java-libs scheduler.build build-test-app

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

binaries=$(shell find . -name "main.go" -exec dirname {} \; |  cut -d/ -f2 | sort | uniq | grep -Ev "vendor|integration|acceptance|test-app")
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

test: autoscaler.test scheduler.test test-acceptance-unit ## Run all unit tests

autoscaler.test: check-db_type init-db test-certs generate-fakes
	@echo ' - using DBURL=${DBURL} TEST=${TEST}'
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' go run github.com/onsi/ginkgo/v2/ginkgo -p ${GINKGO_OPTS} ${TEST} --skip-package='integration,acceptance'

test-autoscaler-suite: check-db_type init-db test-certs build-gorouterproxy
	@echo " - using DBURL=${DBURL} TEST=${TEST}"
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' go run github.com/onsi/ginkgo/v2/ginkgo -p ${GINKGO_OPTS} ${TEST}

test-acceptance-unit:
	@make --directory=acceptance test-unit

.PHONY: build-gorouterproxy
build-gorouterproxy:
	@echo "# building gorouterproxy"
	@CGO_ENABLED=1 go build $(BUILDTAGS) $(BUILDFLAGS) -o build/gorouterproxy integration/gorouterproxy/main.go

.PHONY: integration
integration: generate-fakes init-db test-certs build_all build-gorouterproxy
	@echo "# Running integration tests"
	APP_AUTOSCALER_TEST_RUN='true' DBURL='${DBURL}' go run github.com/onsi/ginkgo/v2/ginkgo ${GINKGO_OPTS} integration DBURL="${DBURL}"

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
lint: generate-fakes
	readonly GOVERSION='${GO_VERSION}' ;\
	export GOVERSION ;\
	echo "Linting with Golang $${GOVERSION}" ;\
	golangci-lint run --config='.golangci.yaml' ${OPTS}

clean-dbtasks:
	pushd dbtasks; mvn clean; popd

package-dbtasks:
	pushd dbtasks; mvn --quiet package ${MVN_OPTS}; popd

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

.PHONY: release-draft
release-draft: ## Create a draft GitHub release with artifacts
	@./scripts/release-autoscaler.sh

.PHONY: cf-login
cf-login:
	@echo '⚠️ Please note that this login only works for cf and concourse,' \
		  'in spite of performing a login as well on bosh and credhub.' \
		  'The necessary changes to the environment get lost when make exits its process.'
	@${MAKEFILE_DIR}/scripts/os-infrastructure-login.sh


.PHONY: start-db
start-db: check-db_type target/start-db-${db_type}_CI_${CI} waitfor_${db_type}_CI_${CI}
	@echo " SUCCESS"


.PHONY: waitfor_postgres_CI_false waitfor_postgres_CI_true
target/start-db-postgres_CI_false:
	@if [ ! "$(shell docker ps -q -f name="^${db_type}")" ]; then \
		if [ "$(shell docker ps -aq -f status=exited -f name="^${db_type}")" ]; then \
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
	else echo " - $@ already up'"; fi;
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

# This target is defined here rather than directly in the component “scheduler” itself, because it depends on targets outside that component. In the future, it will be moved back to that component and reference a dependency to a Makefile on the same level – the one for the component it depends on.
.PHONY: scheduler.test
scheduler.test: check-db_type scheduler.test-certificates init-db
	@make --directory=scheduler test

.PHONY: scheduler.test-certificates
scheduler.test-certificates:
	make --directory=scheduler test-certificates

.PHONY: lint-go
lint-go: lint acceptance.lint test-app.lint


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


.PHONY: deploy-autoscaler deploy-register-cf deploy-autoscaler-bosh deploy-cleanup

deploy-register-cf:
	DEBUG="${DEBUG}" ./scripts/register-broker.sh

deploy-cleanup:
	DEBUG="${DEBUG}" ./scripts/cleanup-autoscaler.sh

deploy-apps:
	DEBUG="${DEBUG}" ./scripts/deploy-apps.sh
