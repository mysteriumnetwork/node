#!/bin/bash

if [ ! -f .env ]; then
    printf "\e[0;31m%s\e[0m\n" "Environment file must be set!"
    exit 1
fi
source .env

TRAVIS_COMMIT="$(git rev-parse --short HEAD 2> /dev/null | sed "s/\(.*\)/\1/")"
TRAVIS_BRANCH="$(git symbolic-ref --short -q HEAD)"
