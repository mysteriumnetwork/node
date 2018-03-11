FROM golang:1.9.2 AS builder

# Compile application
WORKDIR /go/src/github.com/mysterium/node
ADD . .
RUN bin/server_build



FROM frolvlad/alpine-glibc
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Install packages
RUN apk update \
    && apk add --no-cache bash iptables ca-certificates openvpn \
    && rm -rf /var/cache/apk/*

# Install application
COPY --from=builder /go/src/github.com/mysterium/node/build/server/mysterium_server /usr/bin/mysterium_server
COPY bin/server_package/config /etc/mysterium-node

COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
