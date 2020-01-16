FROM alpine:3.6

RUN apk add --update --no-cache iptables ca-certificates openvpn bash sudo openresolv

COPY ./bin/helpers/prepare-run-env.sh /usr/local/bin/prepare-run-env.sh
COPY ./bin/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY ./bin/package/config/common /etc/mysterium-node
COPY ./bin/package/config/linux /etc/mysterium-node

COPY ./build/myst/myst /usr/bin/myst

WORKDIR /var/run/myst

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
