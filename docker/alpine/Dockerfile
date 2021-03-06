FROM postgres:9.6.1-alpine
MAINTAINER <miaolizhao@126.com>

ENV smartcooly_VERSION 0.1.2
RUN apk add --no-cache --virtual .build-deps wget gnupg tar ca-certificates && \
    update-ca-certificates && \
    wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub && \
    wget -q https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.23-r3/glibc-2.23-r3.apk && \
    apk add glibc-2.23-r3.apk && \
    wget -q https://github.com/marstau/smartcooly/releases/download/v${smartcooly_VERSION}/smartcooly_linux_amd64.tar.gz && \
    tar -C /usr/src -xzf smartcooly_linux_amd64.tar.gz && \
    echo $'#!/bin/sh\n\
nohup /docker-entrypoint.sh postgres >/dev/null 2>&1 &\n\
sleep 30s\n\
cd /usr/src/smartcooly_linux_amd64/\n\
./smartcooly\n' >> /usr/src/smartcooly_linux_amd64/cmd.sh && \
    chmod +x /usr/src/smartcooly_linux_amd64/smartcooly && \
    chmod +x /usr/src/smartcooly_linux_amd64/cmd.sh && \
    rm -f smartcooly_linux_amd64.tar.gz glibc-2.23-r3.apk /root/.gnupg && \
    apk del .build-deps && \
    apk add --no-cache --virtual ca-certificates && \
    update-ca-certificates

EXPOSE 9876

VOLUME /var/lib/postgresql/data

COPY config.ini /usr/src/smartcooly_linux_amd64/custom/config.ini

CMD ["/usr/src/smartcooly_linux_amd64/cmd.sh"]
