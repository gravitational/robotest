TARGETS := e2e suite
DOCKER_REPO := quay.io/gravitational
# An empty version will result in build failure, call targets from the top level Makefile to set this
DOCKER_VERSION ?=
# An empty tag will be ignored
DOCKER_TAG ?=
DOCKER_ARGS ?= --pull

GRAVITY_VERSION := 5.5.20
TERRAFORM_VERSION := 0.12.9
CHROMEDRIVER_VERSION := 2.39
TERRAFORM_PROVIDER_AWS_VERSION := 1.19.0
TERRAFORM_PROVIDER_AZURERM_VERSION := 1.5.0
TERRAFORM_PROVIDER_GOOGLE_VERSION := 2.15.0
TERRAFORM_PROVIDER_RANDOM_VERSION := 2.2.0
TERRAFORM_PROVIDER_TEMPLATE_VERSION := 2.1.2
export

providers := AZURERM AWS GOOGLE RANDOM TEMPLATE
provider_args := $(foreach provider,$(providers),--build-arg TERRAFORM_PROVIDER_$(provider)_VERSION=$$TERRAFORM_PROVIDER_$(provider)_VERSION)

BUILD_ARGS := \
	--build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
	--build-arg GRAVITY_VERSION=$(GRAVITY_VERSION) \
	$(provider_args)

E2E_BUILD_ARGS := $(BUILD_ARGS) \
	--build-arg CHROMEDRIVER_VERSION=$(CHROMEDRIVER_VERSION)
SUITE_BUILD_ARGS := $(BUILD_ARGS)

.PHONY: containers
containers: $(TARGETS)

BINARIES := $(addprefix ../build/robotest-,$(TARGETS))

$(BINARIES):
	cd .. && $(MAKE) -j build

.PHONY: $(TARGETS)
$(TARGETS): $(BINARIES)
	$(eval TEMPDIR = "$(shell mktemp -d)")
	$(eval IMAGE = $(DOCKER_REPO)/robotest-$@:$(DOCKER_VERSION))
	if [ -z "$(TEMPDIR)" ]; then \
	  echo "TEMPDIR is not set"; exit 1; \
	fi;
	mkdir -p $(TEMPDIR)/build
	cp -r ../assets/terraform $(TEMPDIR)
	cp -a ../build/robotest-$@ $(TEMPDIR)/build/
	cp -r $@/* $(TEMPDIR)/
	if [ "$@" = "e2e" ]; then \
	  cd $(TEMPDIR) && docker build $(E2E_BUILD_ARGS) --rm=true $(DOCKER_ARGS) -t $(IMAGE) . ; \
	else \
	  cd $(TEMPDIR) && docker build $(SUITE_BUILD_ARGS) --rm=true $(DOCKER_ARGS) -t $(IMAGE) . ; \
	fi
	rm -rf $(TEMPDIR)
	@echo Built $(IMAGE)

# Publish

DOCKER_IMG = $(addprefix $(DOCKER_REPO)/robotest-,$(TARGETS))

.PHONY: publish
# publishing e2e disabled until we find priority for https://github.com/gravitational/robotest/issues/162
# reenable all of DOCKER_IMG as publish deps to begin publishing robotest-e2e again
# - 2020-05 walt
# publish: $(DOCKER_IMG)
publish: $(DOCKER_REPO)/robotest-suite

.PHONY: $(DOCKER_IMG)
$(DOCKER_IMG): $(TARGETS)
	docker push $@:$(DOCKER_VERSION)
ifneq ($(DOCKER_TAG),)
	docker tag $@:$(DOCKER_VERSION) $@:$(DOCKER_TAG)
	docker push $@:$(DOCKER_TAG)
endif


