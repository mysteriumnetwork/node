#!/bin/bash

# Colorful output

SUCCESS_COLOR='\033[0;32m' # green
FAILURE_COLOR="\033[0;31m" # red
DEFAULT_COLOR="\033[0m"

print_success () {
    echo -e $SUCCESS_COLOR$1$DEFAULT_COLOR
}

print_error () {
    echo -e $FAILURE_COLOR$1$DEFAULT_COLOR
}
