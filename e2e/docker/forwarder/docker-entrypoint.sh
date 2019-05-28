#!/bin/bash
set -e

iptables -t nat -A POSTROUTING -o `ip r get ${GATEWAY} | awk '{ print $3 }'` -j MASQUERADE
named -c /etc/bind/named.conf -g -u named
