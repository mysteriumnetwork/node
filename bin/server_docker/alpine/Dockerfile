FROM golang:1.9.2-alpine AS builder

# Install packages
RUN apk add --update --no-cache bash musl-dev gcc make \
    && rm -rf /var/cache/apk/*

# Compile application
WORKDIR /go/src/github.com/mysteriumnetwork/node
ADD . .
RUN bin/server_build



FROM alpine:3.6

# Install packages
RUN apk add --update --no-cache iptables ca-certificates openvpn bash sudo \
    && rm -rf /var/cache/apk/*

COPY bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

COPY bin/server_package/config /etc/mysterium-node
COPY bin/common_package /etc/mysterium-node

COPY --from=builder /go/src/github.com/mysteriumnetwork/node/build/server/mysterium_server /usr/bin/mysterium_server

WORKDIR /var/run/mysterium-node
