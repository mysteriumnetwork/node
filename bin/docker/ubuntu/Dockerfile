FROM anjmao/mysterium-node-builder AS builder

ARG BUILD_BRANCH=$BUILD_BRANCH
ARG BUILD_COMMIT=$BUILD_COMMIT
ARG BUILD_NUMBER=$BUILD_NUMBER
ARG BUILD_VERSION=$BUILD_VERSION
ARG PACKAGE_VERSION

# Compile application
WORKDIR /go/src/github.com/mysteriumnetwork/node
ADD . .

RUN GOOS=linux GOARCH=amd64 bin/build \
    && bin/package_debian ${PACKAGE_VERSION} amd64


FROM ubuntu:16.04

# Install packages
RUN apt-get update \
    && apt-get install -y curl software-properties-common \
    && curl -s https://swupdate.openvpn.net/repos/repo-public.gpg | apt-key add \
    && echo "deb http://build.openvpn.net/debian/openvpn/stable xenial main" > /etc/apt/sources.list.d/openvpn-aptrepo.list \
    && add-apt-repository ppa:wireguard/wireguard \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/*

# Make resolvconf skip the post install (see https://github.com/moby/moby/issues/1297)
RUN apt-get update \
    && apt-get -y install debconf-utils \
    && echo resolvconf resolvconf/linkify-resolvconf boolean false | debconf-set-selections \
    && apt-get -y install resolvconf \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/*

COPY bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY bin/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Install application
COPY --from=builder /go/src/github.com/mysteriumnetwork/node/build/package/myst_linux_amd64.deb /tmp/myst.deb
RUN apt-get update \
    && dpkg --install --force-depends /tmp/myst.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/myst.deb
