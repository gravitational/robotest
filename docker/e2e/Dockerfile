FROM quay.io/gravitational/debian-grande:0.0.1

ARG TERRAFORM_VERSION
ARG CHROMEDRIVER_VERSION

RUN apt-get update && \
    apt-get install -y curl && \
    curl "https://dl-ssl.google.com/linux/linux_signing_key.pub" | apt-key add - && \
    echo 'deb http://dl.google.com/linux/chrome/deb/ stable main' >> /etc/apt/sources.list.d/google.list && \
    apt-get update && \
    apt-get -y install google-chrome-stable xvfb unzip && \
    curl http://chromedriver.storage.googleapis.com/${CHROMEDRIVER_VERSION}/chromedriver_linux64.zip -o chromedriver_linux64.zip && \
    curl https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip -o terraform_linux_amd64.zip && \
    unzip chromedriver_linux64.zip && \
    unzip terraform_linux_amd64.zip && \
    mv chromedriver /usr/bin && \
    mv terraform /usr/bin/ && \
    chmod +x /usr/bin/chromedriver /usr/bin/terraform && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /usr/share/{doc,doc-base,man}/ \
        /tmp/* \
        /chromedriver_linux64.zip \
        /terraform_linux_amd64.zip

RUN adduser chromedriver --uid=995 --disabled-password --system

RUN mkdir -p /robotest
WORKDIR /robotest
COPY entrypoint.sh /entrypoint.sh
COPY build/robotest-e2e /usr/bin/robotest-e2e

RUN chmod +x /usr/bin/robotest-e2e

ENTRYPOINT ["/entrypoint.sh"]
