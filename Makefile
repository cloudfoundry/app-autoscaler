SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c ${SHELLFLAGS}
MVN_OPTS="-Dmaven.test.skip=true"
OS:=$(shell . /etc/lsb-release &>/dev/null && echo $${DISTRIB_ID} ||  uname  )
db_type:=postgres
DBURL := $(shell case "${db_type}" in\
			 (postgres) printf "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"; ;; \
 			 (mysql) printf "root@tcp(localhost)/autoscaler?tls=false"; ;; esac)
CI?=false
$(shell mkdir -p target)

.PHONY: check-type
check-db_type:
	@case "${db_type}" in\
	 (mysql|postgres) echo "Using bd:${db_type}"; ;;\
	 (*) echo "ERROR: db_type needs to be one of mysql|postgres"; exit 1;;\
	 esac

.PHONY: init-db
init-db: check-db_type start-db build-db target/init-db-${db_type}
target/init-db-${db_type}:
	@./scripts/initialise_db.sh ${db_type}
	@touch $@

.PHONY: githooks
githooks: target/githooks
target/githooks: target/precommit-${OS}
	@precommit install
	@touch $@
target/precommit-Darwin:
	@which pre-commit &> /dev/null || brew install pre-commit
	@touch $@
target/precommit-Ubuntu:
	@echo " - not installing $@"
target/precommit-Linux:
	@echo " - not installing $@"

.PHONY: install-cli
install-cli: target/install-cli-${OS}
target/install-cli-Darwin:
	@bosh --version &> /dev/null || brew install cloudfoundry/tap/bosh-cli
	@cf --version &> /dev/null || brew install cloudfoundry/tap/cf-cli@7
	@bbl --version &> /dev/null || brew install bbl
	@[ $(shell cf plugins | grep AutoScaler | wc -l) -gt 1  ] || cf install-plugin -f -r CF-Community app-autoscaler-plugin
	@touch $@
target/install-cli-Ubuntu:
	@echo " - not installing $@"
target/install-cli-Linux:
	@echo " - not installing $@"

.PHONY: init
init: target/init
target/init:
	@make -C src/autoscaler buildtools
	@touch $@

.PHONY: clean
clean:
	@make stop-db db_type=mysql
	@make stop-db db_type=postgres
	@mvn clean > /dev/null
	@make -C src/autoscaler clean
	@rm target/* &> /dev/null || echo "# Already clean"

.PHONY: build-db
build-db: target/build-db
target/build-db:
	@mvn --no-transfer-progress package -pl db ${MVN_OPTS}
	@touch $@

.PHONY: scheduler
scheduler: target/scheduler
target/scheduler:
	@mvn --no-transfer-progress package -pl scheduler ${MVN_OPTS}
	@touch $@

.PHONY: autoscaler
autoscaler: init target/autoscaler
target/autoscaler:
	@make -C src/autoscaler build
	@touch $@

.PHONY: test-certs
test-certs: target/autoscaler_test_certs target/scheduler_test_certs
target/autoscaler_test_certs:
	@./scripts/generate_test_certs.sh
	@touch $@
target/scheduler_test_certs:
	@./scheduler/scripts/generate_unit_test_certs.sh
	@touch $@

.PHONY: test test-autoscaler test-scheduler
test: test-autoscaler test-scheduler
test-autoscaler: check-db_type init init-db test-certs
	@echo " - using DBURL=${DBURL}"
	@make -C src/autoscaler test DBURL="${DBURL}"
test-scheduler: check-db_type init init-db test-certs
	@mvn test --no-transfer-progress -Dspring.profiles.include=${db_type}

start-db: check-db_type target/start-db-${db_type}_CI_${CI} waitfor_${db_type}_CI_${CI}

.PHONY: waitfor_postgres_CI_false waitfor_postgres_CI_true
target/start-db-postgres_CI_false:
	@if [ ! "$(shell docker ps -q -f name=${db_type})" ]; then \
		if [ "$(shell docker ps -aq -f status=exited -f name=${db_type})" ]; then \
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
			postgres:9.6 >/dev/null;\
	else echo " - $@ already up'"; fi;
	@touch $@
target/start-db-postgres_CI_true:
	@echo " - $@ already up'"
waitfor_postgres_CI_false:
	@until docker exec postgres pg_isready &>/dev/null ; do echo " - Waiting for docker postgres"; sleep 1; done
waitfor_postgres_CI_true:
	@echo " - no ci postgres checks"

.PHONY: waitfor_mysql_CI_false waitfor_mysql_CI_true
target/start-db-mysql_CI_false:
	@if [  ! "$(shell docker ps -q -f name=${db_type})" ]; then \
		if [ "$(shell docker ps -aq -f status=exited -f name=${db_type})" ]; then \
			docker rm ${db_type}; \
		fi;\
		echo " - starting docker for ${db_type}";\
		docker run -p 3306:3306  --name mysql \
			-e MYSQL_ALLOW_EMPTY_PASSWORD=true \
			-e MYSQL_DATABASE=autoscaler \
			-d \
			mysql:latest \
			>/dev/null;\
	else echo " - $@ already up"; fi;
	@touch $@
target/start-db-mysql_CI_true:
	@echo " - $@ already up'"
waitfor_mysql_CI_false:
	@until docker exec mysql mysqladmin ping &>/dev/null ; do echo " - Waiting for mysql"; sleep 1; done
	@until [[ ! -z `docker exec mysql mysql -qfsBe "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='autoscaler'"` ]]; do echo " - Waiting for mysql table creation"; sleep 1; done
waitfor_mysql_CI_true:
	@which mysql >/dev/null && until [[ ! -z $(shell mysql -u "root" -h `hostname` --port=3306 -qfsBe "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='autoscaler'") ]]; do echo " - Waiting for mysql table creation"; sleep 1; done

.PHONY: stop
stop-db: check-db_type
	@echo " - Stopping ${db_type}"
	@rm target/start-db-${db_type} &> /dev/null || echo " - Seems the make target was deleted stopping anyway!"
	@docker rm -f ${db_type} &> /dev/null || echo " - we could not stop and remove docker named '${db_type}'"

.PHONY: build
build: init init-db test-certs scheduler autoscaler

.PHONY: integration
integration: build
	make -C src/autoscaler integration DBURL="${DBURL}"