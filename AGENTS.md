# AGENTS.md - Istruzioni per Agenti

## Regola fondamentale: TESTARE DOPO OGNI MODIFICA

Dopo **ogni** modifica al codice (bugfix, nuovo endpoint, refactoring, modifica ai servizi):

1. **Eseguire `go build ./cmd/opaneld` e `go vet ./...`** per verificare che compili senza errori
2. **Rebuild Docker:** `docker stop opanel; docker rm opanel; docker build -t opanel .`
3. **Riavviare il container:** `docker run -d -p 8443:8443 --name opanel opanel`
4. **Rieseguire TUTTI i test esistenti** per verificare che nulla sia rotto (regressione)
5. **Aggiungere nuovi test** per ogni funzionalità aggiunta

### Test completi (eseguire tutti ogni volta)

```
# 1. Health
curl -s http://localhost:8443/api/health

# 2. Login
curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}'

# 3. Login fallito
curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrong"}'

# 4. GetMe
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/auth/me

# 5. Users - List, Create, Get by ID, Update, Delete, Self-delete protection
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users
curl -s -X POST http://localhost:8443/api/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"testuser","email":"test@test.com","password":"Test123!","role":"user"}'
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users/{id}
curl -s -X PUT http://localhost:8443/api/users/{id} -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"email":"new@test.com"}'
curl -s -X DELETE http://localhost:8443/api/users/{id} -H "Authorization: Bearer $TOKEN"
curl -s -X DELETE http://localhost:8443/api/users/1 -H "Authorization: Bearer $TOKEN"  # deve fallire

# 6. Domains - List, Create, Get, Update status, Delete, duplicate check, empty name
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains
curl -s -X POST http://localhost:8443/api/domains -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"testsite.com"}'
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains/{id}
curl -s -X PUT http://localhost:8443/api/domains/{id} -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"suspended"}'
curl -s -X PUT http://localhost:8443/api/domains/{id} -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"active"}'
curl -s -X DELETE http://localhost:8443/api/domains/{id} -H "Authorization: Bearer $TOKEN"

# 7. Databases - List, Create, Get, Delete, duplicate, empty name
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases
curl -s -X POST http://localhost:8443/api/databases -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"testdb"}'
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases/{id}
curl -s -X DELETE http://localhost:8443/api/databases/{id} -H "Authorization: Bearer $TOKEN"

# 8. Database Users - Create, Update password, Update privileges, Delete
curl -s -X POST http://localhost:8443/api/databases/{id}/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"dbuser","password":"Pass123!","privileges":"ALL PRIVILEGES"}'
curl -s -X PUT http://localhost:8443/api/databases/{id}/users/{userId} -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"password":"NewPass123!"}'
curl -s -X PUT http://localhost:8443/api/databases/{id}/users/{userId} -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"privileges":"SELECT"}'
curl -s -X DELETE http://localhost:8443/api/databases/{id}/users/{userId} -H "Authorization: Bearer $TOKEN"

# 9. Verifica servizi Docker
docker exec opanel mysql -u root -e "SHOW DATABASES;"  # MariaDB - deve contenere i DB creati
docker exec opanel mysql -u root -e "SELECT User, Host FROM mysql.user WHERE User NOT IN ('root','mariadb.sys','mysql');"  # Utenti MariaDB
docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'dbuser'@'%';"  # Privilegi utente DB
docker exec opanel cat /etc/nginx/sites-enabled/testsite.com.conf  # Nginx - config generata
docker exec opanel cat /etc/php/8.4/fpm/pool.d/testsite.com.pool.conf  # PHP-FPM - pool generato
docker exec opanel ls /var/www/vhosts/testsite.com/  # Directory vhost (httpdocs, logs, tmp)

# 10. Verifica integrazione PHP-FPM con Domain
# Creare dominio -> pool creato in /etc/php/8.4/fpm/pool.d/{name}.pool.conf
# Eliminare dominio -> pool rimosso

# 11. Verifica che REVOKE funziona prima di GRANT
# Creare utente con ALL PRIVILEGES, poi aggiornare a SELECT -> MariaDB deve mostrare solo SELECT

# 12. Verifica ChangePassword funziona
# Aggiornare password -> testare login MariaDB con nuova password
docker exec opanel mysql -u dbuser -p'NewPass123!' testdb -e "SELECT 1;"  # deve riuscire

# 13. Test completo PowerShell (27 test - da eseguire dopo ogni modifica)
# Copiare e incollare tutto il blocco qui sotto:
$TOKEN = (curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}' | ConvertFrom-Json).token; $pass = 0; $fail = 0; function test($name, $result, $expected) { if ($result -match $expected) { Write-Host "[PASS] $name"; $script:pass++ } else { Write-Host "[FAIL] $name - got: $result"; $script:fail++ } }; test "Health" (curl -s http://localhost:8443/api/health) '"status":"ok"'; test "Login" $TOKEN "eyJhbG"; test "Login wrong" (curl -s -X POST http://localhost:8443/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrong"}') '"error"'; test "GetMe" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/auth/me) '"username":"admin"'; test "Users list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users) '\['; $cu = curl -s -X POST http://localhost:8443/api/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"regtest","email":"reg@test.com","password":"Test123!","role":"user"}'; test "User create" $cu '"id":'; $uid = ($cu | ConvertFrom-Json).id; test "User get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users/$uid) '"username":"regtest"'; test "User update" (curl -s -X PUT http://localhost:8443/api/users/$uid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"email":"new@test.com"}') '"email":"new@test.com"'; test "User delete" (curl -s -X DELETE http://localhost:8443/api/users/$uid -H "Authorization: Bearer $TOKEN") '"message":"user deleted"'; test "User self-delete" (curl -s -X DELETE http://localhost:8443/api/users/1 -H "Authorization: Bearer $TOKEN") '"cannot delete yourself"'; test "Domains list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains) '\['; $cd = curl -s -X POST http://localhost:8443/api/domains -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"regtest.com"}'; test "Domain create" $cd '"id":'; $did = ($cd | ConvertFrom-Json).id; test "Domain get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains/$did) '"name":"regtest.com"'; test "Domain suspend" (curl -s -X PUT http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"suspended"}') '"status":"suspended"'; test "Domain activate" (curl -s -X PUT http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":"active"}') '"status":"active"'; test "Domain delete" (curl -s -X DELETE http://localhost:8443/api/domains/$did -H "Authorization: Bearer $TOKEN") '"id":'; test "Pool removed" (docker exec opanel ls /etc/php/8.4/fpm/pool.d/regtest.com.pool.conf 2>&1) "No such file"; test "DB list" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases) '\['; $cdb = curl -s -X POST http://localhost:8443/api/databases -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"regtest_db"}'; test "DB create" $cdb '"id":'; $dbid = ($cdb | ConvertFrom-Json).id; test "DB get" (curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases/$dbid) '"name":"regtest_db"'; $cdu = curl -s -X POST http://localhost:8443/api/databases/$dbid/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"username":"reg_user","password":"RegP@ss1","privileges":"ALL PRIVILEGES"}'; test "DB user create" $cdu '"id":'; $duid = ($cdu | ConvertFrom-Json).id; test "DB user update pwd" (curl -s -X PUT http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"password":"NewRegP@ss2"}') '"id":'; test "DB user update priv" (curl -s -X PUT http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"privileges":"SELECT"}') '"privileges":"SELECT"'; test "MariaDB grants" (docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'reg_user'@'%';" 2>&1) "GRANT SELECT ON"; test "DB user delete" (curl -s -X DELETE http://localhost:8443/api/databases/$dbid/users/$duid -H "Authorization: Bearer $TOKEN") '"message"'; test "DB delete" (curl -s -X DELETE http://localhost:8443/api/databases/$dbid -H "Authorization: Bearer $TOKEN") '"message"'; test "DB gone in MariaDB" (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'regtest_db';" 2>&1) ""; Write-Host "`n=== RESULTS: $pass PASS, $fail FAIL ==="
```

### Quando aggiungere nuovi test

- **Nuovo endpoint API**: aggiungere test per happy path, errori, validazione input
- **Nuovo servizio**: testare integrazione con Docker (MariaDB, Nginx, PHP-FPM)
- **Nuova tabella migration**: verificare che il DB venga creato correttamente
- **Nuova configurazione**: testare che i valori default funzionino

### Note sulla piattaforma

- Container Docker basato su Debian Trixie (slim)
- PHP 8.4, MariaDB (Debian default), Nginx
- PHP-FPM pool dir: `/etc/php/8.4/fpm/pool.d`
- MariaDB socket: `/var/run/mysqld/mysqld.sock`
- Go 1.24 (go.mod), `CGO_ENABLED=0`
- Le DDL MariaDB (CREATE USER, ALTER USER) usano `fmt.Sprintf` con escape manuale (non parametrizzabili)
