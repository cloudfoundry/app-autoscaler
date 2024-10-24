MAKEFILE_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

PHONY: set-security-group
set-security-group:
	$(eval ORG := $(shell cf target |grep "org\:" |cut -d':' -f2 | xargs))
	$(eval SPACE := $(shell cf target |grep "space\:" |cut -d':' -f2 | xargs))

	cf create-security-group metricsforwarder $(MAKEFILE_DIR)/security-group.json
	cf update-security-group metricsforwarder $(MAKEFILE_DIR)/security-group.json
	cf bind-security-group metricsforwarder $(ORG)
