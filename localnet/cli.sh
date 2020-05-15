#!/bin/bash

set -e

exec /node/build/myst/myst \
  --localnet \
  --transactor.channel-implementation=0x599d43715DF3070f83355D9D90AE62c159E62A75 \
  --transactor.registry-address=0xbe180c8CA53F280C7BE8669596fF7939d933AA10 \
  --accountant.accountant-id=0x7621a5E6EC206309f8E703A653f03F7C8a3097a8 \
  cli
