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

RUN apt-get update && \
    apt-get install -y curl unzip gnupg2 dirmngr

ARG GRAVITY_VERSION
ARG TERRAFORM_VERSION
ARG TERRAFORM_PROVIDER_AZURERM_VERSION
ARG TERRAFORM_PROVIDER_AWS_VERSION
ARG TERRAFORM_PROVIDER_GOOGLE_VERSION
ARG TERRAFORM_PROVIDER_TEMPLATE_VERSION
ARG TERRAFORM_PROVIDER_RANDOM_VERSION
ENV TF_TARBALL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip

ENV TF_PLUGINS \
    # AWS
    https://releases.hashicorp.com/terraform-provider-aws/${TERRAFORM_PROVIDER_AWS_VERSION}/terraform-provider-aws_${TERRAFORM_PROVIDER_AWS_VERSION}_linux_amd64.zip \
    # Azure
    https://releases.hashicorp.com/terraform-provider-azurerm/${TERRAFORM_PROVIDER_AZURERM_VERSION}/terraform-provider-azurerm_${TERRAFORM_PROVIDER_AZURERM_VERSION}_linux_amd64.zip \
    # Google Compute Engine
    https://releases.hashicorp.com/terraform-provider-google/${TERRAFORM_PROVIDER_GOOGLE_VERSION}/terraform-provider-google_${TERRAFORM_PROVIDER_GOOGLE_VERSION}_linux_amd64.zip \
    https://releases.hashicorp.com/terraform-provider-template/${TERRAFORM_PROVIDER_TEMPLATE_VERSION}/terraform-provider-template_${TERRAFORM_PROVIDER_TEMPLATE_VERSION}_linux_amd64.zip \
    https://releases.hashicorp.com/terraform-provider-random/${TERRAFORM_PROVIDER_RANDOM_VERSION}/terraform-provider-random_${TERRAFORM_PROVIDER_RANDOM_VERSION}_linux_amd64.zip

RUN curl ${TF_TARBALL} -o terraform.zip && \
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
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/*

RUN (curl https://get.gravitational.io/telekube/install/${GRAVITY_VERSION} | bash)

RUN mkdir /robotest
WORKDIR /robotest
COPY build/robotest-suite /usr/bin/robotest-suite
COPY terraform /robotest/terraform
COPY run_suite.sh /usr/bin/run_suite.sh

RUN chmod +x /usr/bin/robotest-suite
