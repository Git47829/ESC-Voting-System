#!/bin/bash
set -e

echo "========================================="
echo "   MySQL Network Binding Diagnostic Test"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    local test=$1
    local status=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test"
    else
        echo -e "${RED}✗${NC} $test"
    fi
}

# Function to run test
run_test() {
    local test_name=$1
    local command=$2

    echo -n "Testing: $test_name... "
    if eval "$command" > /tmp/test_output.txt 2>&1; then
        print_status "$test_name" "PASS"
        return 0
    else
        print_status "$test_name" "FAIL"
        echo "Error output:"
        cat /tmp/test_output.txt | sed 's/^/  /'
        return 1
    fi
}

echo "Step 1: Verify containers are running..."
echo ""

if ! docker ps | grep -q escmysql; then
    echo -e "${RED}✗ MySQL container (escmysql) is not running${NC}"
    exit 1
fi
print_status "MySQL container running" "PASS"

if ! docker ps | grep -q db-crud-api; then
    echo -e "${RED}✗ API container (db-crud-api) is not running${NC}"
    exit 1
fi
print_status "API container running" "PASS"

echo ""
echo "Step 2: Check MySQL configuration..."
echo ""

# Check if my.cnf is in the container
run_test "my.cnf exists in container" "docker exec escmysql test -f /etc/mysql/conf.d/docker-mysql.cnf"

# Show bind-address setting
echo -n "Checking bind-address setting... "
BIND_ADDR=$(docker exec escmysql bash -c 'grep -A 5 "\[mysqld\]" /etc/mysql/conf.d/docker-mysql.cnf | grep bind-address' 2>/dev/null || echo "NOT FOUND")
echo "$BIND_ADDR"

echo ""
echo "Step 3: Check MySQL network listener..."
echo ""

# Check if MySQL is listening (from inside container)
echo -n "MySQL listening on port 3306 (from inside container)... "
if docker exec escmysql netstat -tlnp 2>/dev/null | grep -q ":3306"; then
    print_status "MySQL listening" "PASS"
    echo "  Details:"
    docker exec escmysql netstat -tlnp 2>/dev/null | grep ":3306" | sed 's/^/    /'
else
    print_status "MySQL listening" "FAIL"
    echo "  Trying with ss instead..."
    docker exec escmysql ss -tlnp 2>/dev/null | grep ":3306" || echo "    No listeners found"
fi

echo ""
echo "Step 4: Test TCP connection..."
echo ""

# Test from API container to MySQL
echo -n "TCP connection from API to MySQL (mysql:3306)... "
if docker exec db-crud-api bash -c 'timeout 5 bash -c "</dev/tcp/mysql/3306" 2>/dev/null'; then
    print_status "TCP connection" "PASS"
else
    print_status "TCP connection" "FAIL"
fi

echo ""
echo "Step 5: Test MySQL client connection..."
echo ""

# Test actual MySQL connection from API
echo -n "MySQL client connection... "
if docker exec db-crud-api mysql -h mysql -u esc_user -pesc_password -e "SELECT 1;" >/dev/null 2>&1; then
    print_status "MySQL client" "PASS"
else
    echo -e "${RED}✗ Failed${NC}"
    echo "  Trying with root user..."
    if docker exec db-crud-api mysql -h mysql -u root -psecretroot -e "SELECT 1;" >/dev/null 2>&1; then
        echo "  Root connection works - issue may be with esc_user"
    fi
fi

echo ""
echo "Step 6: Check API logs..."
echo ""

echo "Recent API connection attempts:"
docker logs db-crud-api 2>/dev/null | grep "Database connection attempt" | tail -5 | sed 's/^/  /'

echo ""
echo "========================================="
echo "Diagnostic test complete"
echo "========================================="
