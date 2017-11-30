FROM ubuntu:16.04
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# e.g. PACKAGE_FILE="build/package/mysterium-node_linux_amd64.deb"
ARG PACKAGE_FILE
COPY $PACKAGE_FILE /tmp/mysterium-node.deb

# Install packages
RUN apt-get update \
    && dpkg --install --force-depends /tmp/mysterium-node.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/mysterium-node.deb

COPY bin/server_docker/ubuntu/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]