TARGETS := e2e suite
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox
TAG ?= latest
PULL ?= --pull
GLIDE_VER := v0.12.3

# Rules below run on host

.PHONY: build
build: buildbox
	mkdir -p build
	docker run $(DOCKERFLAGS) $(BUILDBOX) make -j $(TARGETS)

.PHONY: all
all: clean build

.PHONY: buildbox
buildbox:
	docker build $(PULL) --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) --build-arg GID=$$(id -g) --build-arg GLIDE_VER=$(GLIDE_VER) \
		docker/build

.PHONY: containers
containers:
	$(MAKE) -C docker containers PULL=$(PULL)
.PHONY: publish
publish:
	$(MAKE) -C docker -j publish TAG=$(TAG)

#
# Runs inside build container
#

.PHONY: $(TARGETS)
$(TARGETS): vendor
	@go version
	cd $(SRCDIR) && \
		go test -c -i ./$(subst robotest-,,$@) -o build/robotest-$@

vendor: glide.yaml
	rm -rf ./.glide ./vendor
	cd $(SRCDIR) && glide install

.PHONY: clean
clean:
	@rm -rf $(BUILDDIR)/* .glide vendor

.PHONY: test
test:
	docker run $(DOCKERFLAGS) $(BUILDBOX) go test -cover -race -v ./infra/...
