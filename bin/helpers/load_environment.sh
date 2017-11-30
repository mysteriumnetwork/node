#!/bin/bash

if [ ! -f .env ]; then
    printf "\e[0;31m%s\e[0m\n" "Environment file must be set!"
    exit 1
fi
source .env