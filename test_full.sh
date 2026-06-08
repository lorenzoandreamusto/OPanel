#!/bin/bash
BASE="http://localhost:8443/api"
TOKEN=$(curl -s -X POST -H "Content-Type: application/json" -d '{"username":"admin","password":"1HBZbdawJ5i5dMFso1FW"}' "$BASE/auth/login" | sed 's/.*"token":"\([^"]*\)".*/\1/')

echo "=== 1. Create domain ==="
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"test.example.com"}' "$BASE/domains"
echo ""

echo "=== 2. List domains ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE/domains"
echo ""

echo "=== 3. Get domain 1 ==="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE/domains/1"
echo ""

echo "=== 4. Check Linux user ==="
id op_test.example.com 2>&1 || echo "user not found"

echo "=== 5. Check vhost dir ==="
ls -la /var/www/vhosts/test.example.com/ 2>&1 || echo "dir not found"

echo "=== 6. Check nginx config ==="
ls /etc/nginx/sites-enabled/ 2>&1 || echo "no sites-enabled"

echo "=== 7. Check nginx -t ==="
nginx -t 2>&1

echo "=== 8. Delete domain ==="
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" "$BASE/domains/1"
echo ""

echo "=== 9. Verify deletion ==="
id op_test.example.com 2>&1 || echo "user removed"
ls /var/www/vhosts/test.example.com/ 2>&1 || echo "dir removed"

echo "=== DONE ==="
