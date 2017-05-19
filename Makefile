TARGETS := e2e suite
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox


GLIDE_VER := v0.12.3

# Amazon S3
BUILD_BUCKET_URL := s3://clientbuilds.gravitational.io/gravity/$(PUBLISH_VERSION)
S3_OPTS := --region us-east-1

# Rules below run on host

.PHONY: all
all: clean build

.PHONY: build
build: buildbox
	mkdir -p build
	docker run $(DOCKERFLAGS) $(BUILDBOX) make -j $(TARGETS)

.PHONY: buildbox
buildbox:
	docker build --pull --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) --build-arg GID=$$(id -g) --build-arg GLIDE_VER=$(GLIDE_VER) \
		docker/build

.PHONY: publish
publish: docker-images publish-binary-into-s3

.PHONY: publish-docker-images
publish-docker-images:
	cd docker && $(MAKE) -j publish-images

.PHONY: publish-binary-into-s3
publish-binary-into-s3:
ifeq (, $(shell which aws))
	$(error "No aws command in $(PATH)")
endif
	aws $(S3_OPTS) s3 cp ./build $(BUILD_BUCKET_URL) --recursive

#
# Runs inside build container
#

.PHONY: $(TARGETS)
$(TARGETS): clean 
	cd $(SRCDIR) && \
		go test -c -i ./$(subst robotest-,,$@) -o build/robotest-$@

.PHONY: deps
deps:
	cd $(SRCDIR) && glide install

.PHONY: clean
clean:
	@rm -rf $(BUILDDIR)/*

.PHONY: test
test:
	docker run $(DOCKERFLAGS) $(BUILDBOX) go test -cover -race -v ./infra/...
