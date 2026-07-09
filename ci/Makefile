.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

repository-root := $(shell realpath --relative-to $$(pwd) $$(git rev-parse --show-toplevel))

# All files that together define the development-environment:
development-environment-definition := \
	$(wildcard ${repository-root}/nix/**/*) \
	${repository-root}/devbox.json \
	${repository-root}/devbox.lock

# 🚸 Overwrite the following parameters to work with a different repository on a different registry:
OCI_REGISTRY ?= ghcr.io
OCI_REPOSITORY ?= cloudfoundry/app-autoscaler-release-tools
OCI_TAG ?= main

.PHONY: docker-login
docker-login:
	@logged_in=$$(jq '.auths | has("${OCI_REGISTRY}")' ~/.docker/config.json)
	readonly logged_in
	if [[ "$${logged_in}" != 'true' ]]
	then
		docker login '${OCI_REGISTRY}'
	else
		echo 'Already logged in into ${OCI_REGISTRY}, skipping login.'
	fi

.PHONY: set-infrastructure-pipeline
set-infrastructure-pipeline:
	@./infrastructure/set-pipeline.sh

.PHONY: unpause-pipeline
unpause-pipeline:
	@./scripts/unpause-pipeline.sh

.PHONY: delete-pipeline
delete-pipeline:
	@./scripts/delete-pipeline.sh

.PHONY: lint
lint:
	@shellcheck --external-sources --shell='bash' **/scripts/*.sh

.PHONY: fix-lint
fix-lint:
	@shellcheck --external-sources --shell='bash' -f diff **/scripts/*.sh

.PHONY: browse
browse:
	@open https://concourse.app-runtime-interfaces.ci.cloudfoundry.org
