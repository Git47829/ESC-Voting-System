#!/bin/bash
# Custom entrypoint to ensure MySQL binds to all interfaces

set -e

# Source the original MySQL entrypoint
. /usr/local/bin/docker-entrypoint.sh

# Ensure bind-address is set to 0.0.0.0
exec mysqld --bind-address=0.0.0.0 --skip-name-resolve "$@"
