PR_NUMBER ?= $(shell gh pr view --json number --jq '.number')
DEPLOYMENT_NAME ?= autoscaler-$(PR_NUMBER)
VM_NAME := $(shell bosh -d $(DEPLOYMENT_NAME) vms --json | jq '.Tables | .[] | .Rows | .[] | select(.instance|test("metricsforwarder")) | .instance')
METRICSFORWARDER_HOST ?= "metricsforwarder.autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"

.PHONY: cf-push
cf-push: fetch-config build-cf
	cf push metricsforwarder -b binary_buildpack -p . --no-start --route ${METRICSFORWARDER_HOST}
	cf start metricsforwarder

.PHONY: fetch-config
fetch-config:
	# how to define variables in deployment name
	mkdir -p assets/certs/policy_db assets/certs/storedprocedure_db assets/certs/syslog_client

	$(eval POSTGRES_IP := $(shell bosh -d ${DEPLOYMENT_NAME} vms --json | jq -r '.Tables | .[] | .Rows  | .[] | select(.instance|test("postgres")) | .ips' ))
	echo "Postgres IP: $(POSTGRES_IP)"

	$(eval LOG_CACHE_IP := $(shell bosh -d cf vms --json | jq -r '.Tables | .[] | .Rows  | .[] | select(.instance|test("log-cache")) | .ips' ))
	echo "LOG_CACHE IP: $(LOG_CACHE_IP)"


	@echo "Pulling metricforwarder config from $(VM_NAME)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/metricsforwarder.yml assets/metricsforwarder.yml

	@echo "Pulling policy db certs from $(VM_NAME)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/ca.crt assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/crt    assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/key    assets/certs/policy_db/.

	@echo "Pulling storeprocedure db certs from $(VM_NAME)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/ca.crt assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/crt	 assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/key    assets/certs/storedprocedure_db/.

	@echo "Pulling syslog-client certs from $(VM_NAME)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/ca.crt		assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.crt assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(VM_NAME):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.key assets/certs/syslog_client/.

	@echo "Build metricsforwarder config yaml"
	cp assets/metricsforwarder.yml metricsforwarder.yml

	sed -i'' -e 's|\/var\/vcap\/jobs\/metricsforwarder\/config|\/home\/vcap\/app/assets|g' metricsforwarder.yml
	sed -i'' -e 's|$(DEPLOYMENT_NAME).autoscalerpostgres.service.cf.internal|$(POSTGRES_IP)|g' metricsforwarder.yml


PHONY: set-security-group
set-security-group:
	$(eval ORG := $(shell cf target |grep "org\:" |cut -d':' -f2 | xargs))
	$(eval SPACE := $(shell cf target |grep "space\:" |cut -d':' -f2 | xargs))

	cf create-security-group metricsforwarder security-group.json
	cf bind-security-group metricsforwarder $(ORG)

PHONY: build-cf
build-cf:
	@cd ../; make build-cf-metricsforwarder

PHONY: stop-metricsforwarder-vm
stop-metricsforwarder-vm:
	$(eval VM_NAME := $(shell bosh -d $(DEPLOYMENT_NAME) vms --json | jq '.Tables | .[] | .Rows | .[] | select(.instance|test("metricsforwarder")) | .instance'))
	bosh -d $(DEPLOYMENT_NAME) stop $(VM_NAME)


