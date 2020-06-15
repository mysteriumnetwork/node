#!/bin/bash

set -e

exec /node/build/myst/myst \
  --config-dir=/etc/mysterium-node \
  --script-dir=/etc/mysterium-node \
  --log-dir= --data-dir=/var/lib/mysterium-node \
  --runtime-dir=/var/run/mysterium-node \
  --tequilapi.address=0.0.0.0 \
  --log-level=debug \
  --payments.mystscaddress=0x4D1d104AbD4F4351a0c51bE1e9CA0750BbCa1665 \
  --ip-detector=http://ipify:3000/?format=json \
  --location.type=manual \
  --broker-address=broker \
  --api.address=http://mysterium-api:8001/v1 \
  --ether.client.rpc=ws://ganache:8545 \
  --keystore.lightweight \
  --transactor.channel-implementation=0x599d43715DF3070f83355D9D90AE62c159E62A75 \
  --transactor.registry-address=0x9a4D866Cb86877f9E51d4C63Bf7fdAf815A980BC \
  --accountant.accountant-id=0x0464a8750d728c4f34F175BD47D6B865a9c0332b \
  --accountant.address=http://accountant:8889/api/v2 \
  --transactor.address=http://transactor:8888/api/v1 \
  --quality.address=http://morqa:8085/api/v1 \
  daemon
