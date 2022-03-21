FROM golang:1.18-alpine

# Install packages
RUN apk add --no-cache iptables ipset ca-certificates openvpn wireguard-tools bash sudo openresolv gcc musl-dev make linux-headers vim curl tcpdump

COPY ./bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY ./bin/package/config/common /etc/mysterium-node
COPY ./bin/package/config/linux /etc/mysterium-node
COPY ./localnet/node/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

WORKDIR /node

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
