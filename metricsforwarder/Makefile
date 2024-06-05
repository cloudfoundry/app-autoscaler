PR_NUMBER ?= $(shell gh pr view --json number --jq '.number')
DEPLOYMENT_NAME ?= autoscaler-$(PR_NUMBER)
METIRCSFORWARDER_VM := $(shell bosh -d $(DEPLOYMENT_NAME) vms --json | jq '.Tables | .[] | .Rows | .[] | select(.instance|test("metricsforwarder")) | .instance')
METRICSFORWARDER_APPNAME ?= "metricsforwarder"
METRICSFORWARDER_HOSTNAME ?= $(METRICSFORWARDER_APPNAME)
SYSTEM_DOMAIN ?= "autoscaler.app-runtime-interfaces.ci.cloudfoundry.org"

.PHONY: cf-push
cf-push: fetch-config build-cf set-security-group
	cf push $(METRICSFORWARDER_APPNAME) -b binary_buildpack -p . --no-start --no-route -c "./metricsforwarder -c metricsforwarder.yml"
	cf create-route  "$(SYSTEM_DOMAIN)" --hostname "$(METRICSFORWARDER_HOSTNAME)"
	cf map-route "$(METRICSFORWARDER_APPNAME)" "$(SYSTEM_DOMAIN)" --hostname "$(METRICSFORWARDER_HOSTNAME)"
	cf start "$(METRICSFORWARDER_APPNAME)"
	@$(MAKE) stop-metricsforwarder-vm

.PHONY: fetch-config
fetch-config: start-metricsforwarder-vm
	# how to define variables in deployment name
	mkdir -p assets/certs/policy_db assets/certs/storedprocedure_db assets/certs/syslog_client

	$(eval POSTGRES_IP := $(shell bosh -d ${DEPLOYMENT_NAME} vms --json | jq -r '.Tables | .[] | .Rows  | .[] | select(.instance|test("postgres")) | .ips' ))
	echo "Postgres IP: $(POSTGRES_IP)"

	$(eval LOG_CACHE_IP := $(shell bosh -d cf vms --json | jq -r '.Tables | .[] | .Rows  | .[] | select(.instance|test("log-cache")) | .ips' ))
	echo "LOG_CACHE IP: $(LOG_CACHE_IP)"


	@echo "Pulling metricforwarder config from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/metricsforwarder.yml assets/metricsforwarder.yml

	@echo "Pulling policy db certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/ca.crt assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/crt    assets/certs/policy_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/policy_db/key    assets/certs/policy_db/.

	@echo "Pulling storeprocedure db certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/ca.crt assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/crt	 assets/certs/storedprocedure_db/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/storedprocedure_db/key    assets/certs/storedprocedure_db/.

	@echo "Pulling syslog-client certs from $(METIRCSFORWARDER_VM)..."
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/ca.crt		assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.crt assets/certs/syslog_client/.
	bosh -d $(DEPLOYMENT_NAME) scp $(METIRCSFORWARDER_VM):/var/vcap/jobs/metricsforwarder/config/certs/syslog_client/client.key assets/certs/syslog_client/.

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

PHONY: start-metricsforwarder-vm
start-metricsforwarder-vm:
	bosh -d $(DEPLOYMENT_NAME) -n start $(METIRCSFORWARDER_VM)

PHONY: stop-metricsforwarder-vm
stop-metricsforwarder-vm:
	bosh -d $(DEPLOYMENT_NAME) -n stop $(METIRCSFORWARDER_VM)


