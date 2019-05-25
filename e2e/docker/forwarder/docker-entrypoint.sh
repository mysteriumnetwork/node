#!/bin/bash
set -e

iptables -t nat -A POSTROUTING -o `ip r get 172.30.0.1 | awk '{ print $3 }'` -j MASQUERADE
named -c /etc/bind/named.conf -g -u named
