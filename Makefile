TARGETS := e2e suite
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox
TAG ?= latest
DOCKER_ARGS ?= --pull
GOLANGCI_LINT_VER ?= 1.21.0

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
build: buildbox
	mkdir -p build
	docker run $(DOCKERFLAGS) $(BUILDBOX) \
		dumb-init make -j $(TARGETS)

.PHONY: all
all: ## Clean and build.
all: clean build

.PHONY: buildbox
buildbox:
	docker build $(DOCKER_ARGS) --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) \
		--build-arg GID=$$(id -g) \
		--build-arg GOLANGCI_LINT_VER=$(GOLANGCI_LINT_VER) \
		docker/build

.PHONY: containers
containers: ## Build container images.
containers: build lint
	$(MAKE) -C docker containers DOCKER_ARGS=$(DOCKER_ARGS)

.PHONY: publish
publish: ## Publish container images to quay.io.
publish: build lint
	$(MAKE) -C docker -j publish TAG=$(TAG)

.PHONY: clean
clean: ## Remove intermediate build artifacts & cache.
	@rm -rf $(BUILDDIR)/*
	@rm -rf vendor

.PHONY: test
test: ## Run unit tests.
	docker run $(DOCKERFLAGS) \
		--env="GO111MODULE=off" \
		$(BUILDBOX) \
		dumb-init go test -cover -race -v ./infra/...

.PHONY: lint
lint: ## Run static analysis against source code.
lint: buildbox
	docker run $(DOCKERFLAGS) \
		--env="GO111MODULE=off" \
		$(BUILDBOX) dumb-init golangci-lint run \
		--skip-dirs=vendor \
		--timeout=2m
#
# Targets below here run inside the buildbox
#
# These are not intended to be called directly by end users.

.PHONY: $(TARGETS)
$(TARGETS): vendor
	@go version
	cd $(SRCDIR) && \
		GO111MODULE=on go test -mod=vendor -c -i ./$(subst robotest-,,$@) -o build/robotest-$@

vendor: go.mod
	cd $(SRCDIR) && go mod vendor
