#!/bin/bash
set -e

# Run the official MySQL entrypoint
# This initializes the database if needed
/usr/local/bin/docker-entrypoint.sh mysqld --bind-address=0.0.0.0 --skip-name-resolve "$@"
