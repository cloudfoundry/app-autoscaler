PR_NUMBER ?= $(shell gh pr view --json number --jq '.number')
DEPLOYMENT_NAME ?= autoscaler-$(PR_NUMBER)
SYSTEM_DOMAIN ?=autoscaler.app-runtime-interfaces.ci.cloudfoundry.org
METIRCSFORWARDER_VM := $(shell bosh -d $(DEPLOYMENT_NAME) vms --json | jq '.Tables | .[] | .Rows | .[] | select(.instance|test("metricsforwarder")) | .instance')
POSTGRES_ADDRESS := $(DEPLOYMENT_NAME)-postgres.tcp.$(SYSTEM_DOMAIN)
LOG_CACHE_IP := $(shell bosh -d cf vms --json | jq -r '.Tables | .[] | .Rows  | .[] | select(.instance|test("log-cache")) | .ips' )
MAKEFILE_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
POSTGRES_EXTERNAL_PORT := $(or $(PR_NUMBER),5432)


.PHONY: fetch-config
fetch-config: start-metricsforwarder-vm
	# how to define variables in deployment name
	mkdir -p build/assets/certs/policy_db build/assets/certs/storedprocedure_db build/assets/certs/syslog_client

	echo "POSTGRES ADDRESS: $(POSTGRES_ADDRESS)"
	echo "LOG_CACHE IP: $(LOG_CACHE_IP)"

	@echo "Pulling metricforwarder config from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/metricsforwarder.yml build/assets/metricsforwarder.yml

	@echo "Pulling policy db certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/ca.crt build/assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/crt    build/assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/key    build/assets/certs/policy_db/.

	@echo "Pulling storeprocedure db certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/ca.crt build/assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/crt	 build/assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/key    build/assets/certs/storedprocedure_db/.

	@echo "Pulling syslog-client certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/ca.crt		build/assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.crt build/assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.key build/assets/certs/syslog_client/.

	@echo "Build metricsforwarder config yaml"
	cp build/assets/metricsforwarder.yml build/metricsforwarder.yml

	sed -i'' -e 's|\/var\/vcap\/jobs\/metricsforwarder\/config|\/home\/vcap\/app/assets|g' build/metricsforwarder.yml
	sed -i'' -e 's|$(DEPLOYMENT_NAME).autoscalerpostgres.service.cf.internal:5432|$(POSTGRES_ADDRESS):$(POSTGRES_EXTERNAL_PORT)|g' build/metricsforwarder.yml



PHONY: set-security-group
set-security-group:
	$(eval ORG := $(shell cf target |grep "org\:" |cut -d':' -f2 | xargs))
	$(eval SPACE := $(shell cf target |grep "space\:" |cut -d':' -f2 | xargs))

	cf create-security-group metricsforwarder $(MAKEFILE_DIR)/security-group.json
	cf bind-security-group metricsforwarder $(ORG)

PHONY: start-metricsforwarder-vm
start-metricsforwarder-vm:
	bosh -d $(DEPLOYMENT_NAME) -n start $(METIRCSFORWARDER_VM)

PHONY: stop-metricsforwarder-vm
stop-metricsforwarder-vm:
	bosh -d $(DEPLOYMENT_NAME) -n stop $(METIRCSFORWARDER_VM)


