FROM golang:1.11-alpine AS builder

# Install packages
RUN apk add --update --no-cache bash gcc musl-dev make \
    && rm -rf /var/cache/apk/*

# Compile application
WORKDIR /go/src/github.com/mysteriumnetwork/node
ADD . .
RUN bin/client_build



FROM alpine:3.6

# Install packages
RUN apk add --update --no-cache iptables ca-certificates openvpn bash sudo \
    && rm -rf /var/cache/apk/*

COPY bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY bin/client_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

COPY bin/client_package/config/linux /etc/mysterium-client
COPY bin/common_package /etc/mysterium-client

COPY --from=builder /go/src/github.com/mysteriumnetwork/node/build/client/mysterium_client /usr/bin/mysterium_client

WORKDIR /var/run/mysterium-client
