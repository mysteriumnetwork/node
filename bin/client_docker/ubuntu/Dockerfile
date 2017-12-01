FROM ubuntu:16.04
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# e.g. PACKAGE_FILE="build/package/mysterium-client_linux_amd64.deb"
ARG PACKAGE_FILE
COPY $PACKAGE_FILE /tmp/mysterium-client.deb

# Install packages
RUN apt-get update \
    && dpkg --install --force-depends /tmp/mysterium-client.deb \
    && apt-get install -y --fix-broken \
    && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/mysterium-client.deb

ENTRYPOINT ["/usr/bin/mysterium_client", "--node=${NODE}"]