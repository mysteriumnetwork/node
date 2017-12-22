FROM alpine:3.6
MAINTAINER Valdas Petrulis <petrulis.valdas@gmail.com>

# Install packages
RUN apk update \
    && apk add --no-cache iptables ca-certificates openvpn \
    && rm -rf /var/cache/apk/*

# Install application
COPY build/server/mysterium_server /usr/bin/mysterium_server
COPY bin/server_package/config /etc/mysterium-node

COPY bin/server_docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]