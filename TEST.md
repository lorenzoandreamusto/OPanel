# Test OPanel

## Test Locale (consigliato per sviluppo)

### Build e avvio completo (frontend + backend)

```bash
# Build frontend + backend in un colpo solo
make build

# Avvia il server
make run
```

Il server parte su `http://localhost:8443` con il frontend servito direttamente.

### Build separati

```bash
# Solo frontend (hot reload per sviluppo)
make frontend-dev
# Vite dev server su http://localhost:3000, proxy API verso :8443

# Build frontend per produzione (output in ./static/)
make frontend-build

# Solo backend Go
go build -o ./bin/opaneld ./cmd/opaneld
./bin/opaneld server
```

### Verifica TypeScript

```bash
cd frontend && npm run typecheck
```

---

## Test su Docker (simula VPS Debian 13)

### Deploy con Docker Compose (consigliato)

```bash
# Build e avvia
docker compose up -d --build

# Logs
docker compose logs -f

# Ferma
docker compose down
```

### Deploy manuale con Docker

```bash
# Rimuovi container precedenti
docker rm -f opanel 2>/dev/null

# Build image (include frontend)
docker build -t opanel .

# Avvia
docker run -d -p 8443:8443 --name opanel opanel
```

### Deploy da scratch (install.sh)

```bash
# Rimuovi container precedenti
docker rm -f opanel 2>/dev/null

# Avvia Debian 13 pulito
docker run -d -p 8443:8443 --name opanel debian:trixie-slim sleep infinity

# Copia i file dentro
docker cp . opanel:/tmp/opanel

# Installa tutto (Nessun systemd in Docker → niente service/firewall)
docker exec -it opanel bash -c "cd /tmp/opanel && bash install.sh"

# Avvia opaneld (senza systemd, va avviato manualmente)
docker exec -d opanel /opt/opanel/bin/opaneld server --config /etc/opanel/config.yaml
```

---

## Verifica API

```bash
# Health check
curl http://localhost:8443/api/health

# Login
curl -X POST http://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Prendi la password (se usi install.sh)
docker exec opanel cat /etc/opanel/config.yaml
```

## Verifica Frontend

```bash
# La pagina principale deve restituire l'HTML di OPanel
curl -s http://localhost:8443/ | grep "OPanel"

# SPA routing: tutti i path non-API restituiscono index.html
curl -s http://localhost:8443/login | grep "OPanel"
curl -s http://localhost:8443/domains | grep "OPanel"

# Asset statici (CSS/JS)
curl -s -o /dev/null -w "%{http_code}" http://localhost:8443/assets/index-Cy-MvAeB.css
curl -s -o /dev/null -w "%{http_code}" http://localhost:8443/assets/index-cbgErPX1.js
```

## Test completo (27 API + frontend)

```powershell
# PowerShell - copia e incolla tutto
$TOKEN = (curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}' | ConvertFrom-Json).token; $pass = 0; $fail = 0; function test($name, $result, $expected) { if ($result -match $expected) { Write-Host "[PASS] $name"; $script:pass++ } else { Write-Host "[FAIL] $name - got: $result"; $script:fail++ } }; test "Health" (curl -s http://localhost:8443/api/health) '"status":"ok"'; test "Login" $TOKEN "eyJhbG"; test "Login wrong" (curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrong"}') '"error"'; test "GetMe" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/auth/me) '"username":"admin"'; test "Users list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users) '\['; $cu = curl -s -X POST http://localhost:8443/api/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"regtest","email":"reg@test.com","password":"Test123!","role":"user"}'; test "User create" $cu '"id":'; $uid = ($cu | ConvertFrom-Json).id; test "User get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users/$uid) '"username":"regtest"'; test "User update" (curl -s -X PUT http://localhost:8443/api/users/$uid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"email":"new@test.com"}') '"email":"new@test.com"'; test "User delete" (curl -s -X DELETE http://localhost:8443/api/users/$uid -H "Authorization: Bearer $TOKEN") '"message"'; test "User self-delete" (curl -s -X DELETE http://localhost:8443/api/users/1 -H "Authorization: Bearer $TOKEN") '"cannot delete yourself"'; test "Domains list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains) '\['; $cd = curl -s -X POST http://localhost:8443/api/domains -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"regtest.com"}'; test "Domain create" $cd '"id":'; $did = ($cd | ConvertFrom-Json).id; test "Domain get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains/$did) '"name":"regtest.com"'; test "Domain suspend" (curl -s -X PUT http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"suspended"}') '"status":"suspended"'; test "Domain activate" (curl -s -X PUT http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"active"}') '"status":"active"'; test "Pool created" (docker exec opanel ls /etc/php/8.4/fpm/pool.d/regtest.com.pool.conf 2>&1) "regtest.com.pool.conf"; test "Nginx created" (docker exec opanel ls /etc/nginx/sites-enabled/regtest.com.conf 2>&1) "regtest.com.conf"; test "Domain delete" (curl -s -X DELETE http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN") '"id":'; test "Pool removed" (docker exec opanel ls /etc/php/8.4/fpm/pool.d/regtest.com.pool.conf 2>&1) "No such file"; test "DB list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases) '\['; $cdb = curl -s -X POST http://localhost:8443/api/databases -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"regtest_db"}'; test "DB create" $cdb '"id":'; $dbid = ($cdb | ConvertFrom-Json).id; test "DB get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases/$dbid) '"name":"regtest_db"'; $cdu = curl -s -X POST http://localhost:8443/api/databases/$dbid/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"reg_user","password":"RegP@ss1","privileges":"ALL PRIVILEGES"}'; test "DB user create" $cdu '"id":'; $duid = ($cdu | ConvertFrom-Json).id; test "DB user update pwd" (curl -s -X PUT http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"password":"NewRegP@ss2"}') '"id":'; test "DB user update priv" (curl -s -X PUT http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"privileges":"SELECT"}') '"privileges":"SELECT"'; test "MariaDB grants" (docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'reg_user'@'%';" 2>&1) "GRANT SELECT ON"; test "DB user delete" (curl -s -X DELETE http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN") '"message"'; test "DB delete" (curl -s -X DELETE http://localhost:8443/api/databases/$dbid -H "Authorization: Bearer $TOKEN") '"message"'; test "DB gone in MariaDB" (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'regtest_db';" 2>&1) ""; test "Frontend page" (curl -s http://localhost:8443/) "OPanel"; test "SPA /login" (curl -s http://localhost:8443/login) "OPanel"; test "SPA /domains" (curl -s http://localhost:8443/domains) "OPanel"; Write-Host "`n=== RESULTS: $pass PASS, $fail FAIL ==="
```

---

## Cleanup

```bash
# Docker
docker rm -f opanel

# Build locale
make clean
```
