FROM quay.io/gravitational/debian-grande:0.0.1

ARG TERRAFORM_VERSION
ARG CHROMEDRIVER_VERSION
ARG TERRAFORM_AZURERM_VERSION
ARG TERRAFORM_AWS_VERSION

ENV TF_TARBALL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
ENV TF_AWS_TARBALL https://releases.hashicorp.com/terraform-provider-aws/${TERRAFORM_AWS_VERSION}/terraform-provider-aws_${TERRAFORM_AWS_VERSION}_linux_amd64.zip
ENV CHROMEDRIVER_TARBALL http://chromedriver.storage.googleapis.com/${CHROMEDRIVER_VERSION}/chromedriver_linux64.zip
ENV TF_AZURERM_TARBALL https://releases.hashicorp.com/terraform-provider-azurerm/${TERRAFORM_AZURERM_VERSION}/terraform-provider-azurerm_${TERRAFORM_AZURERM_VERSION}_linux_amd64.zip

RUN apt-get update && \
    apt-get install -y curl && \
    curl "https://dl-ssl.google.com/linux/linux_signing_key.pub" | apt-key add - && \
    echo 'deb http://dl.google.com/linux/chrome/deb/ stable main' >> /etc/apt/sources.list.d/google.list && \
    apt-get update && \
    apt-get -y install google-chrome-stable xvfb unzip && \
    curl $TF_TARBALL -o terraform.zip && \
    curl $TF_AWS_TARBALL -o terraform_aws.zip && \
    curl $TF_AZURERM_TARBALL -o terraform_azurerm.zip && \
    curl $CHROMEDRIVER_TARBALL -o chromedriver.zip && \
    unzip terraform.zip -d /usr/bin && \
    mkdir -p /etc/terraform/plugins && \
    unzip terraform_aws.zip -d /etc/terraform/plugins && \
    unzip terraform_azurerm.zip -d /etc/terraform/plugins && \
    unzip chromedriver.zip && \
    mv chromedriver /usr/bin && \
    chmod +x /usr/bin/chromedriver /usr/bin/terraform && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/* \
        /*.zip


RUN adduser chromedriver --uid=995 --disabled-password --system

RUN mkdir -p /robotest
WORKDIR /robotest
COPY entrypoint.sh /entrypoint.sh
COPY build/robotest-e2e /usr/bin/robotest-e2e

RUN chmod +x /usr/bin/robotest-e2e && \
    chmod +x /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
