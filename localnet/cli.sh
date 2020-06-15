#!/bin/bash

set -e

exec /node/build/myst/myst \
  --localnet \
  --transactor.channel-implementation=0x599d43715DF3070f83355D9D90AE62c159E62A75 \
  --transactor.registry-address=0x9a4D866Cb86877f9E51d4C63Bf7fdAf815A980BC \
  --accountant.accountant-id=0x0464a8750d728c4f34F175BD47D6B865a9c0332b \
  cli
