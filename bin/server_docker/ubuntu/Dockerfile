FROM golang:1.9.2 AS builder

ARG PACKAGE_VERSION

# Install FPM
RUN apt-get update \
    && apt-get install -y ruby-dev build-essential \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* \
    && gem install ffi -v 1.9.18 \
    && gem install fpm -v 1.9.3

# Install Debber
RUN go get github.com/debber/debber-v0.3/cmd/debber

# Compile application
WORKDIR /go/src/github.com/mysterium/node
ADD . .
RUN GOOS=linux GOARCH=amd64 bin/server_build \
    && bin/server_package_debian ${PACKAGE_VERSION} amd64



FROM ubuntu:16.04
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>
ENV NODE_LOCATION_COUNTRY ""

# Install packages
COPY --from=builder /go/src/github.com/mysterium/node/build/package/mysterium-node_linux_amd64.deb /tmp/mysterium-node.deb

RUN apt-get update \
    && apt-get install -y curl \
    && curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add \
    && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list \
    && apt-get update \
    && apt-get install -y openvpn ca-certificates iptables \
    && dpkg --install --force-depends /tmp/mysterium-node.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/mysterium-node.deb

COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
