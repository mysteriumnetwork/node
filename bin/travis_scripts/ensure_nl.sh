#!/bin/bash
#
# Check and warn if files do not end in a newline

set -u # Undefined variables are errors
source bin/helpers/output.sh

function main() {
    ensure_newline bin/
}

function ensure_newline() {
    declare -a no_newline
    local dir=$1

    no_newline=$(find ./bin -type f -print0  |
		   xargs -0 -L1 bash -c 'test "$(tail -c 1 "$0")" && echo $0' |
		   grep -v keystore)

    if [ ! -z "$no_newline" ]; then
	print_warning "Some scripts appear to not end in a newline: "
	print_warning "$no_newline"
    fi
}

main $@
