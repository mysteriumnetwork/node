FROM golang:1.9.2-alpine AS builder

# Install packages
RUN apk add --update --no-cache bash g++ make \
    && rm -rf /var/cache/apk/*

# Compile application
WORKDIR /go/src/github.com/mysterium/node
ADD . .
RUN bin/client_build



FROM alpine:3.6
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Install packages
RUN apk add --update --no-cache bash iptables ca-certificates openvpn \
    && rm -rf /var/cache/apk/*

# Directories
RUN mkdir -p /etc/mysterium-client \
    && mkdir -p /var/lib/mysterium-client \
    && mkdir -p /var/run/mysterium-client

COPY bin/client_package/config /etc/mysterium-client
COPY --from=builder /go/src/github.com/mysterium/node/build/client/mysterium_client /usr/bin/mysterium_client

WORKDIR /var/run/mysterium-client

ENTRYPOINT [ "/usr/bin/mysterium_client", \
    "--config-dir=/etc/mysterium-client", \
    "--data-dir=/var/lib/mysterium-client", \
    "--runtime-dir=/var/run/mysterium-client", \
    "--tequilapi.address=0.0.0.0" ]

