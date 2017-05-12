BINARY := robotest
VERSION ?= $(shell git describe --long --tags --always|awk -F'[.-]' '{print $$1 "." $$2 "." $$4}')
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox
GLIDE_VER := v0.12.3

IMAGE_NAME := $(BINARY)
TARBALL_NAME := $(IMAGE_NAME)-$(VERSION).tar
IMAGE := quay.io/gravitational/$(IMAGE_NAME):$(VERSION)
LATEST_IMAGE := quay.io/gravitational/$(IMAGE_NAME):latest
PUBLISH_VERSION := latest

# Amazon S3
BUILD_BUCKET_URL := s3://clientbuilds.gravitational.io/gravity/$(PUBLISH_VERSION)
S3_OPTS := --region us-east-1

.PHONY: all
all: clean build

.PHONY: build
build: buildbox
	mkdir -p build
	docker run $(DOCKERFLAGS) $(BUILDBOX) make $(BINARY)

$(BINARY): clean
	cd $(SRCDIR) && \
		glide install && \
		go test -c -i ./e2e -o build/$(BINARY)

.PHONY: clean
clean:
	@rm -rf $(BUILDDIR)/$(BINARY)
	@rm -f $(TARBALL_NAME)

.PHONY: test
test:
	go test -cover -race -v ./infra/...

buildbox:
	docker build --pull --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) --build-arg GID=$$(id -g) --build-arg GLIDE_VER=$(GLIDE_VER) .

.PHONY: docker-image
docker-image:
	$(eval TEMPDIR = "$(shell mktemp -d)")
	if [ -z "$(TEMPDIR)" ]; then \
	  echo "TEMPDIR is not set"; exit 1; \
	fi;
	mkdir -p $(TEMPDIR)/build
	cp -a build/$(BINARY) $(TEMPDIR)/build/
	cp -r docker/* $(TEMPDIR)/
	cd $(TEMPDIR) && docker build --rm=true --pull -t $(IMAGE) .
	rm -rf $(TEMPDIR)

.PHONY: print-image
print-image:
	echo $(IMAGE)

.PHONY: publish
publish: docker-image publish-image publish-binary-into-s3

.PHONY: publish-image
publish-image:
	docker tag $(IMAGE) $(LATEST_IMAGE)
	docker push $(IMAGE)
	docker push $(LATEST_IMAGE)

.PHONY: publish-binary-into-s3
publish-binary-into-s3:
ifeq (, $(shell which aws))
	$(error "No aws command in $(PATH)")
endif
	aws $(S3_OPTS) s3 cp ./build/robotest $(BUILD_BUCKET_URL)/robotest

.PHONY: what-version
what-version:
	@echo $(VERSION)

