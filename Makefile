PROGRAM := robotest
PREFIX := bin

NOROOT := -u $$(id -u):$$(id -g)
SRCDIR := /go/src/github.com/gravitational/robotest
DOCKERFLAGS := --rm=true $(NOROOT) -v $(PWD):$(SRCDIR) -w $(SRCDIR)
BUILDIMAGE := golang:1.7.1

BINS := $(PREFIX)

.PHONY: all
all: build

.PHONY: build
build:
	docker run $(DOCKERFLAGS) $(BUILDIMAGE) make $(BINS)

$(BINS): clean
	go build -o $(PREFIX)/$(PROGRAM)

.PHONY: clean
clean:
	@rm -rf $(BINS)

.PHONY: test
test:
	go test ./lib/... -cover -race -v

