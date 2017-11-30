FROM ubuntu:16.04
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Which Mysterium VPN package version to build
ARG VERSION

# Install packages
RUN apt-get update \
    && apt install -y --no-install-recommends wget ca-certificates \
    && wget https://github.com/MysteriumNetwork/node/releases/download/${VERSION}/mysterium-node_linux_amd64.deb \
    && dpkg --install --force-depends mysterium-node_linux_amd64.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* \
    && rm mysterium-node_linux_amd64.deb

COPY bin/server_docker/ubuntu/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]