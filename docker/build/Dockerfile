FROM golang:1.9.2

ARG UID
ARG GID
ARG GLIDE_VER

RUN groupadd builder --gid=$GID -o; \
    useradd builder --uid=$UID --gid=$GID --create-home --shell=/bin/bash;

RUN (mkdir -p /go/src/github.com/gravitational/robotest && chown -R builder /go)
RUN (mkdir -p /go/bin)

ENV LANGUAGE="en_US.UTF-8" \
    LANG="en_US.UTF-8" \
    LC_ALL="en_US.UTF-8" \
    LC_CTYPE="en_US.UTF-8" \
    GOPATH="/go" \
    PATH="$PATH:/opt/go/bin:/go/bin"

RUN (wget https://github.com/Masterminds/glide/releases/download/$GLIDE_VER/glide-$GLIDE_VER-linux-amd64.tar.gz && \
	tar -xvf glide-$GLIDE_VER-linux-amd64.tar.gz -C /go/bin linux-amd64/glide --strip-components=1 && \
	rm glide-$GLIDE_VER-linux-amd64.tar.gz)

VOLUME ["/go/src/github.com/gravitational/robotest"]
