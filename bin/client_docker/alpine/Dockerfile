FROM golang:1.9.2-alpine AS builder

# Install packages
RUN apk add --update --no-cache bash gcc musl-dev make \
    && rm -rf /var/cache/apk/*

# Compile application
WORKDIR /go/src/github.com/mysterium/node
ADD . .
RUN bin/client_build



FROM alpine:3.6
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Install packages
RUN apk add --update --no-cache iptables ca-certificates openvpn bash \
    && rm -rf /var/cache/apk/*

# Directories
RUN mkdir -p /var/lib/mysterium-client

COPY bin/client_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

COPY bin/client_package/config /etc/mysterium-client
COPY --from=builder /go/src/github.com/mysterium/node/build/client/mysterium_client /usr/bin/mysterium_client

WORKDIR /var/run/mysterium-client
