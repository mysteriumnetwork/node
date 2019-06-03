FROM golang:1.11-alpine AS builder

# Install packages
RUN apk add --update --no-cache bash gcc musl-dev make conntrack-tools tcpdump \
    && rm -rf /var/cache/apk/*

ARG BUILD_BRANCH=$BUILD_BRANCH
ARG BUILD_COMMIT=$BUILD_COMMIT
ARG BUILD_NUMBER=$BUILD_NUMBER
ARG BUILD_VERSION=$BUILD_VERSION

# Compile application
WORKDIR /go/src/github.com/mysteriumnetwork/node
ADD . .
RUN bin/build


FROM alpine:3.6

# Install packages
RUN apk add --update --no-cache iptables ca-certificates openvpn bash sudo curl \
    && rm -rf /var/cache/apk/*

COPY bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY bin/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

COPY bin/package/config/common /etc/mysterium-node
COPY bin/package/config/linux /etc/mysterium-node

COPY --from=builder /go/src/github.com/mysteriumnetwork/node/build/myst/myst /usr/bin/myst

WORKDIR /var/run/myst
