#!/bin/bash
# Health check script for MySQL
# Tests if MySQL is ready to accept connections on port 3306

# First check if port 3306 is listening
for i in {1..5}; do
  if mysqladmin ping -h 127.0.0.1 --protocol=TCP -u root -psecretroot 2>/dev/null; then
    exit 0
  fi
  sleep 1
done

exit 1
