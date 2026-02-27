#!/bin/bash
# Check what MySQL actually thinks its configuration is

echo "=== MySQL Configuration Check ==="
echo ""

echo "1. What bind-address did MySQL actually load?"
docker exec escmysql mysql -u root -psecretroot -e "SHOW VARIABLES LIKE 'bind_address';" 2>/dev/null

echo ""
echo "2. Full MySQL config files MySQL is reading:"
docker exec escmysql bash -c 'find /etc/mysql -name "*.cnf" -type f 2>/dev/null | xargs ls -la'

echo ""
echo "3. Content of my.cnf (if exists):"
docker exec escmysql cat /etc/mysql/conf.d/docker-mysql.cnf 2>/dev/null || echo "File not found"

echo ""
echo "4. MySQL startup command:"
docker exec escmysql ps aux | grep mysqld | grep -v grep

echo ""
echo "5. MySQL error log (last 50 lines):"
docker exec escmysql tail -50 /var/log/mysql/error.log 2>/dev/null || echo "No error log found"

echo ""
echo "6. Check if port 3306 is being listened to at all:"
docker exec escmysql bash -c 'ss -tlnp 2>/dev/null || netstat -tlnp' | grep -i mysql || echo "No MySQL listeners found"
