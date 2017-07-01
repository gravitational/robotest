FROM quay.io/gravitational/debian-grande:0.0.1

ARG TERRAFORM_VERSION

RUN apt-get update && \
    apt-get install -y curl unzip

RUN    curl https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip -o terraform_linux_amd64.zip && \
    unzip terraform_linux_amd64.zip && \
    mv terraform /usr/bin/ && \
    chmod +x /usr/bin/terraform && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/* \
        /terraform_linux_amd64.zip

RUN mkdir -p /robotest
WORKDIR /robotest
COPY build/robotest-suite /usr/bin/robotest-suite
COPY build/go-junit-report /usr/bin/go-junit-report
COPY terraform /robotest/terraform
COPY run_suite.sh /usr/bin/run_suite.sh

RUN chmod +x /usr/bin/robotest-suite