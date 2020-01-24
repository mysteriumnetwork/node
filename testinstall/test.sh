#!/usr/bin/env bash

shopt -s extglob
set -e
set -o errtrace
set -o errexit
set -o pipefail

script_url=https://raw.githubusercontent.com/mysteriumnetwork/node/master/install.sh

boxes=(
  'buster'
  'stretch'
  'bionic'
  'xenial'
)

for box in "${boxes[@]}"; do
    vagrant up --provision "$box"
    vagrant ssh "$box" -c "curl $script_url | sudo bash" || (echo -e "\e[31mFAIL: $box\e[0m" && exit 1)
    vagrant ssh "$box" -c "curl localhost:4050/healthcheck" || (echo -e "\e[31mFAIL: $box\e[0m" && exit 1)
    vagrant destroy -f "$box" || (echo -e "\e[31mFAIL: $box\e[0m" && exit 1)
    echo -e "\e[32mSUCCESS: $box\e[0m"
done
