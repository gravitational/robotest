FROM quay.io/gravitational/debian-venti:go1.10.3-stretch

ARG UID
ARG GID
ARG GLIDE_VER
ARG GOLANGCI_LINT_VER

RUN groupadd builder --gid=$GID -o; \
    useradd builder --uid=$UID --gid=$GID --create-home --shell=/bin/bash;

RUN (mkdir -p /go/src/github.com/gravitational/robotest && chown -R builder /go)
RUN (mkdir -p /go/bin)

ENV LANGUAGE="en_US.UTF-8" \
    LANG="en_US.UTF-8" \
    LC_ALL="en_US.UTF-8" \
    LC_CTYPE="en_US.UTF-8" \
    GOPATH="/gopath" \
    PATH="$PATH:/opt/go/bin:/go/bin"

RUN (wget -q https://github.com/Masterminds/glide/releases/download/$GLIDE_VER/glide-$GLIDE_VER-linux-amd64.tar.gz && \
	tar -xvf glide-$GLIDE_VER-linux-amd64.tar.gz -C /go/bin linux-amd64/glide --strip-components=1 && \
	rm glide-$GLIDE_VER-linux-amd64.tar.gz) && \
    (wget -q https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VER/golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz && \
	tar -xvf golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz -C /go/bin \
		golangci-lint-$GOLANGCI_LINT_VER-linux-amd64/golangci-lint --strip-components=1 && \
	rm golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz)

 VOLUME ["/gopath/src/github.com/gravitational/robotest"]
