FROM alpine:3.6 AS forwarder

# Install packages
RUN apk add --update --no-cache bash gcc musl-dev make iptables conntrack-tools tcpdump dnsmasq \
    && rm -rf /var/cache/apk/*

COPY e2e/docker/forwarder/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
