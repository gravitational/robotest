TARGETS := e2e suite
DOCKER_REPO := quay.io/gravitational
VERSION ?= $(shell git describe --long --tags --always|awk -F'[.-]' '{print $$1 "." $$2 "." $$4}')
TAG ?= latest
PULL ?= --pull

TERRAFORM_VERSION := 0.11.7
CHROMEDRIVER_VERSION := 2.35
TERRAFORM_AWS_VERSION := 1.19.0
TERRAFORM_AZURERM_VERSION := 1.5.0

E2E_BUILD_ARGS := --build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
	--build-arg CHROMEDRIVER_VERSION=$(CHROMEDRIVER_VERSION) \
	--build-arg TERRAFORM_AZURERM_VERSION=$(TERRAFORM_AZURERM_VERSION) \
	--build-arg TERRAFORM_AWS_VERSION=$(TERRAFORM_AWS_VERSION)

SUITE_BUILD_ARGS := \
	--build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
	--build-arg TERRAFORM_AZURERM_VERSION=$(TERRAFORM_AZURERM_VERSION) \
	--build-arg TERRAFORM_AWS_VERSION=$(TERRAFORM_AWS_VERSION)

.PHONY: containers
containers: $(TARGETS)

BINARIES := $(addprefix ../build/robotest-,$(TARGETS))

$(BINARIES):
	cd .. && $(MAKE) -j build

.PHONY: $(TARGETS)
$(TARGETS): $(BINARIES)
	$(eval TEMPDIR = "$(shell mktemp -d)")
	$(eval IMAGE = $(DOCKER_REPO)/robotest-$@:$(VERSION))
	if [ -z "$(TEMPDIR)" ]; then \
	  echo "TEMPDIR is not set"; exit 1; \
	fi;
	mkdir -p $(TEMPDIR)/build
	cp -r ../assets/terraform $(TEMPDIR)
	cp -a ../build/robotest-$@ $(TEMPDIR)/build/
	cp -r $@/* $(TEMPDIR)/
	if [ "$@" = "e2e" ]; then \
	  cd $(TEMPDIR) && docker build $(E2E_BUILD_ARGS) --rm=true $(PULL) -t $(IMAGE) . ; \
	else \
	  cd $(TEMPDIR) && docker build $(SUITE_BUILD_ARGS) --rm=true $(PULL) -t $(IMAGE) . ; \
	fi
	rm -rf $(TEMPDIR)
	@echo Built $(IMAGE)

# Publish

DOCKER_IMG = $(addprefix $(DOCKER_REPO)/robotest-,$(TARGETS))

.PHONY: publish
publish: $(DOCKER_IMG)

.PHONY: $(DOCKER_IMG)
$(DOCKER_IMG): $(TARGETS)
	docker push $@:$(VERSION)
	docker tag $@:$(VERSION) $@:$(TAG)
	docker push $@:$(TAG)

.PHONY: what-version
what-version:
	@echo $(VERSION)
