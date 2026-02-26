#!/bin/bash
# Health check script for MySQL
# Tests if MySQL is ready to accept connections

# Use mysqladmin to ping with explicit TCP protocol
mysqladmin ping -h 127.0.0.1 --protocol=TCP -u root -psecretroot 2>/dev/null

exit $?
