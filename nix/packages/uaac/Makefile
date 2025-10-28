SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c ${SHELLFLAGS}
.ONESHELL:

.PHONY: gemset.nix
gemset.nix: Gemfile.lock ## Generate the gemset.nix file from the Gemfile.lock
	bundix --lock

help: ## Show this help
	@grep --extended-regexp --no-filename '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

