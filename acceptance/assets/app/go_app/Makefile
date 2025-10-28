.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c ${SHELLFLAGS}
aes_terminal_font_yellow := \e[38;2;255;255;0m
aes_terminal_reset := \e[0m

# TODO: Do we need the next line?
MAKEFLAGS =

GO_VERSION = $(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES = $(shell find . -type f -name '*.go')
PACKAGE_DIRS = $(shell go list './...' | grep --invert-match --regexp='/vendor/' \
								 | grep --invert-match --regexp='e2e')
CGO_ENABLED := 0
export GOWORK := off

binaries = $(shell find . -name "main.go" -exec dirname {} \; \
									 | cut --delimiter='/' --fields='2' | sort | uniq)
test_dirs = $(shell find . -name "*_test.go" -exec dirname {} \; \
										| cut --delimiter='/' --fields='2' | sort | uniq)

GINKGO_OPTS := -r --race --require-suite --randomize-all --cover ${OPTS}
export CONFIG ?= ../../../../acceptance_config.json

openapi-spec-path := ../../../../../api
openapi-specs-list = $(wildcard ${openapi-spec-path}/*.openapi.yaml)

# The presence of the subsequent directory indicates wheather the fakes still need to be generated
# or not.
app-fakes-dir := ./internal/app/appfakes
app-fakes-files = $(wildcard ${app-fakes-dir}/*.go)
go_deps_without_fakes = $(shell find . -type f -name '*.go' \
																| grep --invert-match --regexp='${app-fakes-dir}')

.PHONY: generate-fakes
generate-fakes: ${app-fakes-dir} ${app-fakes-files}
${app-fakes-dir} ${app-fakes-files} &: ./go.mod ./go.sum ./internal/generate.go ${openapi-specs-list}
	go generate ./...



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
go-mod-tidy: ./go.mod ./go.sum ${go_deps_without_fakes}
	@echo -ne '${aes_terminal_font_yellow}'
	@echo -e '⚠️ Warning: The client-fakes generated from the openapi-specification may be\n' \
					 'outdated. Please consider re-generating them, if this is relevant.'
	@echo -ne '${aes_terminal_reset}'
	go mod tidy



.PHONY: build
build: ./build/app ./build/manifest.yml
./build/app ./build/manifest.yml: ./go.mod ${app-fakes-dir} ${app-fakes-files}
	echo '# building  test-app'
	mkdir -p build
	CGO_ENABLED='${CGO_ENABLED}' GOOS='linux' GOARCH='amd64' go build -o './build/app'
	cp './app_manifest.yml' './build/manifest.yml'

.PHONY: build_tests
build_tests: $(addprefix build_test-,$(test_dirs))

build_test-%: ${app-fakes-dir} ${app-fakes-files}
	@echo " - building '$*' tests"
	@export build_folder='${PWD}/build/tests/$*'
	@mkdir -p "$${build_folder}"
	cd $*
	for package in $$(go list './...' | sed 's|.*/autoscaler/$*|.|' | awk '{ print length, $$0 }' | \
										sort --numeric-sort --reverse | cut --delimiter=' ' --fields='2-')
	do
		export test_file="$${build_folder}/$${package}.test"
		echo "   - compiling $${package} to $${test_file}"
		go test -c -o "$${test_file}" "$${package}"
	done


.PHONY: check lint lint-fix test
check: lint build test

test: generate-fakes
	@echo 'Running tests'
	ginkgo run ${GINKGO_OPTS} './...'

lint: generate-fakes
	golangci-lint run

lint-fix: generate-fakes
	golangci-lint run --fix



.PHONY: start
start: build
	docker run --interactive --tty --name go_app --volume="$${PWD}/build/:/cf/bin" \
						 --publish '8080:8080' --rm 'cloudfoundry/cflinuxfs4' '/cf/bin/app'



.PHONY: deploy
deploy: build
	./deploy.sh



.PHONY: clean
clean:
	@echo "# cleaning go_app"
	@go clean -cache -testcache
	@rm --force --recursive './build'
	@rm --force --recursive './internal/app/appfakes'
