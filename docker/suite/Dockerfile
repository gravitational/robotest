FROM quay.io/gravitational/debian-grande:0.0.1

ARG TERRAFORM_VERSION
ARG TERRAFORM_AZURERM_VERSION
ARG TERRAFORM_AWS_VERSION

ENV TF_TARBALL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
ENV TF_AWS_TARBALL https://releases.hashicorp.com/terraform-provider-aws/${TERRAFORM_AWS_VERSION}/terraform-provider-aws_${TERRAFORM_AWS_VERSION}_linux_amd64.zip
ENV TF_AZURERM_TARBALL https://releases.hashicorp.com/terraform-provider-azurerm/${TERRAFORM_AZURERM_VERSION}/terraform-provider-azurerm_${TERRAFORM_AZURERM_VERSION}_linux_amd64.zip

RUN apt-get update && \
    apt-get install -y curl unzip

RUN \
    curl $TF_TARBALL -o terraform.zip && \
    curl $TF_AWS_TARBALL -o terraform_aws.zip && \
    curl $TF_AZURERM_TARBALL -o terraform_azurerm.zip && \
    unzip terraform.zip -d /usr/bin && \
    mkdir -p /etc/terraform/plugins && \
    unzip terraform_aws.zip -d /etc/terraform/plugins && \
    unzip terraform_azurerm.zip -d /etc/terraform/plugins && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/* \
        /terraform.zip \
        /terraform_aws.zip \
        /terraform_azurerm.zip


RUN (curl https://get.gravitational.io/telekube/install | bash)

RUN mkdir /robotest
WORKDIR /robotest
COPY build/robotest-suite /usr/bin/robotest-suite
COPY terraform /robotest/terraform
COPY run_suite.sh /usr/bin/run_suite.sh

RUN chmod +x /usr/bin/robotest-suite
