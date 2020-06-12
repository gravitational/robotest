# Copyright 2020 Gravitational, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
FROM quay.io/gravitational/debian-venti:go1.12.9-stretch

ARG UID
ARG GID
ARG GOLANGCI_LINT_VER

# Output errors from go list
# https://github.com/golang/tools/commit/4a1b41eed170862fe82028699015842e9382fe2b
ENV GOPACKAGESPRINTGOLISTERRORS=true

RUN groupadd builder --gid=$GID -o; \
    useradd builder --uid=$UID --gid=$GID --create-home --shell=/bin/bash;

RUN (mkdir -p /go/src/github.com/gravitational/robotest && chown -R builder /go ${GOPATH})
RUN (mkdir -p /go/bin)

ENV LANGUAGE="en_US.UTF-8" \
    LANG="en_US.UTF-8" \
    LC_ALL="en_US.UTF-8" \
    LC_CTYPE="en_US.UTF-8" \
    GOPATH="/gopath" \
    PATH="$PATH:/opt/go/bin:/go/bin"

RUN (wget -q https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VER/golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz && \
	tar -xvf golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz -C /go/bin \
		golangci-lint-$GOLANGCI_LINT_VER-linux-amd64/golangci-lint --strip-components=1 && \
	rm golangci-lint-$GOLANGCI_LINT_VER-linux-amd64.tar.gz)

WORKDIR /gopath/src/github.com/gravitational/robotest
VOLUME ["/gopath/src/github.com/gravitational/robotest"]
