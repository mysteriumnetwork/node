FROM mysteriumnetwork/mysterium-node-builder AS builder

# Compile application
ADD . .
ARG PACKAGE_VERSION
RUN GOOS=linux GOARCH=amd64 bin/server_build \
    && bin/server_package_debian ${PACKAGE_VERSION} amd64



FROM ubuntu:16.04
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Install packages
RUN apt-get update \
    && apt-get install -y curl \
    && curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add \
    && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/*

COPY bin/server_docker/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Install application
COPY --from=builder /go/src/github.com/mysterium/node/build/package/mysterium-node_linux_amd64.deb /tmp/mysterium-node.deb
RUN apt-get update \
    && dpkg --install --force-depends /tmp/mysterium-node.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/mysterium-node.deb
