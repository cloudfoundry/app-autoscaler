PR_NUMBER := $(shell gh pr view --json number --jq '.number')
DEPLOYMENT_NAME ?= autoscaler-$(PR_NUMBER)
VM_NAME := $(shell bosh -d $(DEPLOYMENT_NAME) vms --json | jq ".Tables | .[0] | .Rows | .[] | .instance" -r | grep metricsforwarder)

fetch-config: cf-login
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

	sed -i ''  's|\/var\/vcap\/jobs\/metricsforwarder\/config|\/home\/vcap\/app/assets|g' metricsforwarder.yml
	sed -i ''  's|$(DEPLOYMENT_NAME).autoscalerpostgres.service.cf.internal|$(POSTGRES_IP)|g' metricsforwarder.yml
	sed -i ''  's|log-cache.service.cf.internal|$(LOG_CACHE_IP)|g' metricsforwarder.yml

set-security-group:
	$(eval ORG := $(shell cf target |grep "org\:" |cut -d':' -f2 | xargs))
	$(eval SPACE := $(shell cf target |grep "space\:" |cut -d':' -f2 | xargs))

	cf create-security-group metricsforwarder security-group.json
	cf bind-security-group metricsforwarder $(ORG)

build-cf:
	@cd ../; make build-cf-metricsforwarder



cf-login:
	@cd ../../../; make cf-login

