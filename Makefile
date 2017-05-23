TARGETS := e2e suite
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
BUILDDIR ?= $(abspath build)
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -v $(BUILDDIR):$(SRCDIR)/build -w $(SRCDIR)
BUILDBOX := robotest:buildbox

GLIDE_VER := v0.12.3

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
publish:
	cd docker && $(MAKE) -j publish

#
# Runs inside build container
#

.PHONY: $(TARGETS)
$(TARGETS): clean vendor
	cd $(SRCDIR) && \
		go test -c -i ./$(subst robotest-,,$@) -o build/robotest-$@

vendor: glide.yaml
	cd $(SRCDIR) && glide install

glide.yaml:

.PHONY: clean
clean:
	@rm -rf $(BUILDDIR)/*

.PHONY: test
test:
	docker run $(DOCKERFLAGS) $(BUILDBOX) go test -cover -race -v ./infra/...
