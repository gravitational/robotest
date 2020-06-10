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
FROM quay.io/gravitational/debian-grande:stretch

ARG TERRAFORM_VERSION
ARG CHROMEDRIVER_VERSION
ARG TERRAFORM_PROVIDER_AWS_VERSION

ENV TF_TARBALL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
ENV TF_PLUGINS \
    # AWS
    https://releases.hashicorp.com/terraform-provider-aws/${TERRAFORM_PROVIDER_AWS_VERSION}/terraform-provider-aws_${TERRAFORM_PROVIDER_AWS_VERSION}_linux_amd64.zip
ENV CHROMEDRIVER_TARBALL http://chromedriver.storage.googleapis.com/${CHROMEDRIVER_VERSION}/chromedriver_linux64.zip

RUN \
    apt-get update && \
    apt-get install -y curl gnupg2 dirmngr && \
    curl "https://dl-ssl.google.com/linux/linux_signing_key.pub" | apt-key add - && \
    echo 'deb http://dl.google.com/linux/chrome/deb/ stable main' >> /etc/apt/sources.list.d/google.list && \
    apt-get update && \
    apt-get -y install google-chrome-stable xvfb unzip && \
    \
    curl $TF_TARBALL -o terraform.zip && \
    curl ${TF_TARBALL} -o terraform.zip && \
    unzip terraform.zip -d /usr/bin && \
    rm -f terraform.zip && \
    mkdir -p /etc/terraform/plugins && \
    \
    for plugin in $TF_PLUGINS; do \
        curl ${plugin} -o plugin.zip && \
        unzip plugin.zip -d /etc/terraform/plugins && \
        rm -f plugin.zip; \
    done && \
    \
    curl $CHROMEDRIVER_TARBALL -o chromedriver.zip && \
    unzip chromedriver.zip && \
    mv chromedriver /usr/bin && \
    chmod +x /usr/bin/chromedriver /usr/bin/terraform && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/*

RUN adduser chromedriver --uid=995 --disabled-password --system

RUN mkdir -p /robotest
WORKDIR /robotest
COPY entrypoint.sh /entrypoint.sh
COPY build/robotest-e2e /usr/bin/robotest-e2e

RUN chmod +x /usr/bin/robotest-e2e && \
    chmod +x /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
