#!/bin/sh -e

# Source debconf library.
. /usr/share/debconf/confmodule

db_input critical mysterium/terms || true
db_go

# to reset accepted terms:
# echo RESET mysterium/accept_terms | debconf-communicate mysterium-node
# or:
db_fset mysterium/accept_terms seen false

db_input critical mysterium/accept_terms || true
db_go

# Check their answer.
db_get mysterium/accept_terms
if [ "$RET" = "false" ]; then
    # terminate installation
    db_purge || true
    echo "You did not accept our terms and conditions. Installation cancelled.\n" >&2
    exit 2
fi
