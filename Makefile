BINARY := e2e.test
NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -w $(SRCDIR)
BUILDBOX := robotest:buildbox
GLIDE_VER := v0.12.3

.PHONY: all
all: build

.PHONY: build
build: buildbox
	docker run $(DOCKERFLAGS) $(BUILDBOX) make $(BINARY)

$(BINARY): clean
	cd $(SRCDIR) && \
		glide install && \
		go test -c -i ./e2e

.PHONY: clean
clean:
	@rm -rf $(BINARY)

.PHONY: test
test:
	go test -cover -race -v ./infra/...

buildbox:
	docker build --pull --tag $(BUILDBOX) \
		--build-arg UID=$$(id -u) --build-arg GID=$$(id -g) --build-arg GLIDE_VER=$(GLIDE_VER) .
