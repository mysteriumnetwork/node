FROM golang:1.9.2-alpine
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

ADD . /go/src/github.com/mysterium/node
WORKDIR /go/src/github.com/mysterium/node

# Install packages
RUN apk update \
    && apk add --no-cache bash iptables openvpn \
    && rm -rf /var/cache/apk/*

# Compile application
RUN bin/server_build \
    && rm -rf /go/src/github.com/mysterium/node/*

# Install application
COPY build/server/mysterium_server /usr/bin/mysterium_server
COPY bin/server_package/config /etc/mysterium-node

COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]