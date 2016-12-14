FROM debian:jessie
RUN apt-get update
RUN apt-get -y install curl

RUN mkdir -p /selenium
WORKDIR /selenium

RUN curl "https://dl-ssl.google.com/linux/linux_signing_key.pub" | apt-key add -
RUN echo 'deb http://dl.google.com/linux/chrome/deb/ stable main' >> /etc/apt/sources.list.d/google.list
RUN apt-get update
RUN apt-get -y install google-chrome-stable xvfb unzip

RUN curl http://chromedriver.storage.googleapis.com/2.24/chromedriver_linux64.zip -o chromedriver_linux64.zip
RUN unzip chromedriver_linux64.zip
RUN mv chromedriver /usr/bin/

RUN adduser selenium --uid=995 --disabled-password --system

EXPOSE 9515
COPY run.sh /selenium
ENTRYPOINT /selenium/run.sh
