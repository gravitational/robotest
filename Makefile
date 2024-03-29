# Copyright 2020 Gravitational, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
TARGETS := e2e suite
ifeq ($(origin VERSION), undefined)
# avoid lazily evaluating version.sh (and thus rerunning the shell command several times)
VERSION := $(shell ./version.sh)
endif
GIT_COMMIT := $(shell git rev-parse HEAD)
VERSIONFILE := version.go
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
# docker doesn't allow "+" in image tags: https://github.com/docker/distribution/issues/1201
export ROBOTEST_DOCKER_VERSION ?= $(subst +,-,$(VERSION))
export ROBOTEST_DOCKER_TAG ?=
export ROBOTEST_DOCKER_ARGS ?= --pull
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox
BUILDBOX_IIDFILE := $(BUILDDIR)/.robotest-buildbox.iid
GOLANGCI_LINT_VER ?= 1.32.2

.PHONY: help
# kudos to https://gist.github.com/prwhite/8168133 for inspiration
help: ## Show this message.
	@echo 'Usage: make [options] [target] ...'
	@echo
	@echo 'Options: run `make --help` for options'
	@echo
	@echo 'Targets:'
	@egrep '^(.+)\:\ ##\ (.+)' ${MAKEFILE_LIST} | column -t -c 2 -s ':#' | sort | sed 's/^/  /'

# Rules below run on host

.PHONY: build
build: ## Compile go binaries.
build: vendor buildbox | $(BUILDDIR)
	docker run $(DOCKERFLAGS) $(BUILDBOX) \
		dumb-init make -j $(TARGETS)

.PHONY: all
all: ## Clean and build.
all: clean build

$(BUILDDIR):
	mkdir -p $(BUILDDIR)

.PHONY: buildbox
buildbox: $(BUILDBOX_IIDFILE)

$(BUILDBOX_IIDFILE): docker/build/Dockerfile | $(BUILDDIR)
	docker build $(DOCKER_ARGS) --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) \
		--build-arg GID=$$(id -g) \
		--build-arg GOLANGCI_LINT_VER=$(GOLANGCI_LINT_VER) \
		--iidfile $(BUILDBOX_IIDFILE) \
		docker/build

.PHONY: containers
containers: ## Build container images.
containers: build lint
	$(MAKE) -C docker containers

.PHONY: publish
publish: ## Publish container images to quay.io.
publish: build lint
	$(MAKE) -C docker -j publish

.PHONY: clean
clean: ## Remove intermediate build artifacts & cache.
	-rm -r $(BUILDDIR)
	-rm -r vendor
	-rm $(VERSIONFILE)

.PHONY: test
test: ## Run unit tests.
test: buildbox
	docker run $(DOCKERFLAGS) \
		$(BUILDBOX) \
		dumb-init go test -cover -race -v ./infra/... ./lib/config/...

.PHONY: lint
lint: ## Run static analysis against source code.
lint: vendor buildbox
	docker run $(DOCKERFLAGS) \
		$(BUILDBOX) dumb-init golangci-lint run

.PHONY: vendor
vendor: ## Download dependencies into vendor directory.
vendor: vendor/modules.txt

vendor/modules.txt: go.mod | $(VERSIONFILE) $(BUILDBOX_IIDFILE)
	docker run $(DOCKERFLAGS) $(BUILDBOX) \
		dumb-init go mod vendor
	@touch vendor/modules.txt

.PHONY: version
version: ## Show the robotest version.
version: $(VERSIONFILE)
	@echo "Robotest Version: $(VERSION)"

define VERSION_GO
/* DO NOT EDIT THIS FILE. IT IS GENERATED BY MAKE */

package robotest

const(
	Version   = "$(VERSION)"
	GitCommit = "$(GIT_COMMIT)"
)
endef

# $(VERSIONFILE) is PHONY because make doesn't know if the `git describe` output
# has changed (new commit or tag).  Thus we generate a new draft every time & compare
# to see if the version changed.
.PHONY: $(VERSIONFILE)
$(VERSIONFILE): export CONTENT=$(VERSION_GO)
$(VERSIONFILE): TMP = $(BUILDDIR)/$(VERSIONFILE).tmp
$(VERSIONFILE): $(BUILDDIR)
	@if [ -z "$(VERSION)" ]; then \
		echo "unable to determine version, please fetch tags"; \
		exit 1; \
	fi
	@if [ -z "$(GIT_COMMIT)" ]; then \
		echo "unable to determine current commit"; \
		exit 1; \
	fi
	@echo "$$CONTENT" > $(TMP)
	@if ! cmp -s $(TMP) $(VERSIONFILE); then \
		echo "$$CONTENT" > $(VERSIONFILE); \
		echo "version metadata saved to $(VERSIONFILE)"; \
	fi
	@rm $(TMP)

#
# Targets below here run inside the buildbox
#
# These are not intended to be called directly by end users.

.PHONY: $(TARGETS)
$(TARGETS): $(VERSIONFILE)
	@go version
	go test -c ./$(subst robotest-,,$@) -o build/robotest-$@
