#!/bin/bash

set -e

exec /node/build/myst/myst \
  --localnet \
  --transactor.channel-implementation=0x599d43715DF3070f83355D9D90AE62c159E62A75 \
  --transactor.registry-address=0xbe180c8CA53F280C7BE8669596fF7939d933AA10 \
  --accountant.accountant-id=0xf2e2c77D2e7207d8341106E6EfA469d1940FD0d8 \
  cli
