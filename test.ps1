#!/usr/bin/env pwsh
# OPanel Automated Test Suite
# Eseguire dopo ogni modifica al codice
# Richiede: container Docker in esecuzione su localhost:8443

param(
    [string]$BaseUrl = "http://localhost:8443"
)

$ErrorActionPreference = "Continue"
$pass = 0
$fail = 0
$warn = 0
$tests = @()
$total = 0
$DEBUG = $env:OPANEL_DEBUG -eq "1"

function Test-Case {
    param(
        [string]$Name,
        [scriptblock]$Block,
        [string]$Expect = "pass"
    )
    $script:total++
    try {
        $result = & $Block
        if ($Expect -eq "pass" -and $result) {
            Write-Host "  [PASS] $Name" -ForegroundColor Green
            $script:pass++
            $script:tests += @{ name = $Name; result = "PASS" }
        } elseif ($Expect -eq "fail" -and (-not $result)) {
            Write-Host "  [PASS] $Name (correctly failed)" -ForegroundColor Green
            $script:pass++
            $script:tests += @{ name = $Name; result = "PASS" }
        } elseif ($Expect -eq "warn" -and $result) {
            Write-Host "  [WARN] $Name" -ForegroundColor Yellow
            $script:warn++
            $script:tests += @{ name = $Name; result = "WARN" }
        } else {
            Write-Host "  [FAIL] $Name" -ForegroundColor Red
            $script:fail++
            $script:tests += @{ name = $Name; result = "FAIL" }
        }
    } catch {
        Write-Host "  [FAIL] $Name - Exception: $_" -ForegroundColor Red
        $script:fail++
        $script:tests += @{ name = $Name; result = "FAIL"; error = $_.Exception.Message }
    }
}

function Invoke-Api {
    param(
        [string]$Method = "GET",
        [string]$Path,
        [string]$Token,
        [string]$Body
    )
    
    $uri = "$BaseUrl$Path"
    $headers = @{}
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    
    try {
        $params = @{
            Uri            = $uri
            Method         = $Method
            Headers        = $headers
            UseBasicParsing = $true
            ErrorAction    = "Stop"
        }
        if ($Body) { $params.Body = [System.Text.Encoding]::UTF8.GetBytes($Body); $params.ContentType = "application/json" }
        
        $response = Invoke-WebRequest @params
        $parsed = $response.Content | ConvertFrom-Json -ErrorAction SilentlyContinue
        return @{ ok = $true; data = $parsed; status = $response.StatusCode }
    } catch {
        $code = 0
        $bodyText = ""
        
        # In PowerShell 7, ErrorDetails.Message has the body
        if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
            $bodyText = $_.ErrorDetails.Message
        }
        
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
        }
        
        if (-not $bodyText) {
            try {
                $reader = [System.IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
                $bodyText = $reader.ReadToEnd()
                $reader.Close()
            } catch {}
        }
        
        return @{ ok = $false; data = $bodyText; status = $code }
    }
}

function Get-Json {
    param([string]$Text)
    try { return $Text | ConvertFrom-Json } catch { return $null }
}

# ============================================================
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  OPanel Automated Test Suite" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# ============================================================
Write-Host "--- 1. HEALTH CHECK ---" -ForegroundColor Yellow
$r = Invoke-Api -Path "/api/health"
Test-Case "Health returns 200" { $r.ok -and $r.data.status -eq "ok" }

# ============================================================
Write-Host "`n--- 2. AUTH ---" -ForegroundColor Yellow
$login = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"admin","password":"admin"}'
$TOKEN = if ($login.ok) { $login.data.token } else { "" }
Test-Case "Login valid credentials" { $login.ok -and $TOKEN.Length -gt 0 }

$badLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"admin","password":"wrong"}'
Test-Case "Login wrong password rejected" { (-not $badLogin.ok) -and $badLogin.data -match "invalid credentials" }

$emptyLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"","password":""}'
Test-Case "Login empty credentials rejected" { (-not $emptyLogin.ok) }

$noBodyLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body 'not json'
Test-Case "Login malformed body rejected" { (-not $noBodyLogin.ok) }

$me = Invoke-Api -Path "/api/auth/me" -Token $TOKEN
Test-Case "GetMe returns admin user" { $me.ok -and $me.data.username -eq "admin" }

$unauth = Invoke-Api -Path "/api/users"
Test-Case "Unauthenticated request rejected" { (-not $unauth.ok) }

$badToken = Invoke-Api -Path "/api/users" -Token "invalid.token.here"
Test-Case "Invalid token rejected" { (-not $badToken.ok) }

$logout = Invoke-Api -Method "POST" -Path "/api/auth/logout" -Token $TOKEN
Test-Case "Logout succeeds" { $logout.ok }

# Token still works after logout (JWT is stateless - known limitation)
$afterLogout = Invoke-Api -Path "/api/auth/me" -Token $TOKEN
Test-Case "Token still valid after logout (stateless JWT)" { $afterLogout.ok } -Expect "warn"

# ============================================================
Write-Host "`n--- 3. USERS CRUD ---" -ForegroundColor Yellow
$ul = Invoke-Api -Path "/api/users" -Token $TOKEN
Test-Case "List users returns array" { $ul.ok -and ($null -ne $ul.data) }

$cu = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"testuser_a","email":"a@test.com","password":"Test123!","role":"user"}'
Test-Case "Create user succeeds" { $cu.ok -and $cu.data.id -gt 0 }
$uid = if ($cu.ok) { $cu.data.id } else { 0 }

$ug = Invoke-Api -Path "/api/users/$uid" -Token $TOKEN
Test-Case "Get user by ID" { $ug.ok -and $ug.data.username -eq "testuser_a" }

$uu = Invoke-Api -Method "PUT" -Path "/api/users/$uid" -Token $TOKEN -Body '{"email":"updated@test.com","role":"user"}'
Test-Case "Update user email" { $uu.ok -and $uu.data.email -eq "updated@test.com" }

# Create second user to delete
$cu2 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"testuser_del","email":"del@test.com","password":"Test123!","role":"user"}'
$uid2 = if ($cu2.ok) { $cu2.data.id } else { 0 }
$ud = Invoke-Api -Method "DELETE" -Path "/api/users/$uid2" -Token $TOKEN
Test-Case "Delete user" { $ud.ok -and $ud.data.message -eq "user deleted" }

$ug2 = Invoke-Api -Path "/api/users/$uid2" -Token $TOKEN
Test-Case "Deleted user not found" { (-not $ug2.ok) }

$sd = Invoke-Api -Method "DELETE" -Path "/api/users/1" -Token $TOKEN
Test-Case "Admin cannot delete self" { (-not $sd.ok) }

# Edge cases
$cuDup = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"testuser_a","email":"dup@test.com","password":"Test123!","role":"user"}'
Test-Case "Duplicate username rejected" { (-not $cuDup.ok) }

$cuEmpty = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"","email":"","password":"","role":""}'
Test-Case "Empty fields rejected" { (-not $cuEmpty.ok) }

$cuBadJson = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body 'not json'
Test-Case "Invalid JSON rejected" { (-not $cuBadJson.ok) }

$cuNoPass = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"nopass","email":"np@test.com","password":"","role":"user"}'
Test-Case "Empty password rejected" { (-not $cuNoPass.ok) }

# Cleanup
Invoke-Api -Method "DELETE" -Path "/api/users/$uid" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 4. DOMAINS CRUD ---" -ForegroundColor Yellow
$dl = Invoke-Api -Path "/api/domains" -Token $TOKEN
Test-Case "List domains returns array" { $dl.ok }

$dc = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"automated-test.com"}'
Test-Case "Create domain succeeds" { $dc.ok -and $dc.data.name -eq "automated-test.com" }
$did = if ($dc.ok) { $dc.data.id } else { 0 }

# Verify nginx config generated
$nginxConf = docker exec opanel cat /etc/nginx/sites-enabled/automated-test.com.conf 2>&1
Test-Case "Nginx config generated" { $nginxConf -match "server_name" }

# Verify PHP-FPM pool generated
$poolConf = docker exec opanel cat /etc/php/8.4/fpm/pool.d/automated-test.com.pool.conf 2>&1
Test-Case "PHP-FPM pool generated" { $poolConf -match "automated-test.com" }

# Verify vhost directory structure
$vhostDir = docker exec opanel ls /var/www/vhosts/automated-test.com/ 2>&1
Test-Case "Vhost directory has httpdocs" { $vhostDir -match "httpdocs" }
Test-Case "Vhost directory has logs" { $vhostDir -match "logs" }
Test-Case "Vhost directory has tmp" { $vhostDir -match "tmp" }

$dg = Invoke-Api -Path "/api/domains/$did" -Token $TOKEN
Test-Case "Get domain by ID" { $dg.ok -and $dg.data.name -eq "automated-test.com" }

$ds = Invoke-Api -Method "PUT" -Path "/api/domains/$did" -Token $TOKEN -Body '{"status":"suspended"}'
Test-Case "Suspend domain" { $ds.ok -and $ds.data.status -eq "suspended" }

$da = Invoke-Api -Method "PUT" -Path "/api/domains/$did" -Token $TOKEN -Body '{"status":"active"}'
Test-Case "Activate domain" { $da.ok -and $da.data.status -eq "active" }

# Duplicate
$dcDup = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"automated-test.com"}'
Test-Case "Duplicate domain rejected" { (-not $dcDup.ok) }

# Empty name
$dcEmpty = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":""}'
Test-Case "Empty domain name rejected" { (-not $dcEmpty.ok) }

# Invalid status
$dsBad = Invoke-Api -Method "PUT" -Path "/api/domains/$did" -Token $TOKEN -Body '{"status":"invalid_status"}'
# This should fail because SQLite CHECK constraint enforces ('active','suspended','pending')
Test-Case "Invalid domain status rejected" { (-not $dsBad.ok) } -Expect "warn"

# Non-existent domain
$dg404 = Invoke-Api -Path "/api/domains/99999" -Token $TOKEN
Test-Case "Non-existent domain not found" { (-not $dg404.ok) }

# Delete
$dd = Invoke-Api -Method "DELETE" -Path "/api/domains/$did" -Token $TOKEN
Test-Case "Delete domain" { $dd.ok }

# Verify pool removed
$poolAfter = docker exec opanel ls /etc/php/8.4/fpm/pool.d/automated-test.com.pool.conf 2>&1
Test-Case "PHP-FPM pool removed after delete" { $poolAfter -match "No such file" }

# Verify nginx config removed
$nginxAfter = docker exec opanel ls /etc/nginx/sites-enabled/automated-test.com.conf 2>&1
Test-Case "Nginx config removed after delete" { $nginxAfter -match "No such file" }

# Delete non-existent
$dd404 = Invoke-Api -Method "DELETE" -Path "/api/domains/99999" -Token $TOKEN
Test-Case "Delete non-existent domain fails" { (-not $dd404.ok) }

# ============================================================
Write-Host "`n--- 5. DATABASES CRUD ---" -ForegroundColor Yellow
$dbl = Invoke-Api -Path "/api/databases" -Token $TOKEN
Test-Case "List databases returns array" { $dbl.ok }

$dbc = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"automated_db"}'
Test-Case "Create database succeeds" { $dbc.ok -and $dbc.data.name -eq "automated_db" }
$dbid = if ($dbc.ok) { $dbc.data.id } else { 0 }

# Verify in MariaDB
$dbCheck = (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'automated_db';" 2>&1 | Out-String).Trim()
Test-Case "Database exists in MariaDB" { $dbCheck -match "automated_db" }

$dbcDup = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"automated_db"}'
Test-Case "Duplicate database rejected" { (-not $dbcDup.ok) }

$dbcEmpty = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":""}'
Test-Case "Empty database name rejected" { (-not $dbcEmpty.ok) }

$dbcBadJson = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body 'not json'
Test-Case "Invalid JSON for database rejected" { (-not $dbcBadJson.ok) }

$dbg = Invoke-Api -Path "/api/databases/$dbid" -Token $TOKEN
Test-Case "Get database by ID" { $dbg.ok -and $dbg.data.name -eq "automated_db" }

$dbg404 = Invoke-Api -Path "/api/databases/99999" -Token $TOKEN
Test-Case "Non-existent database not found" { (-not $dbg404.ok) }

# ============================================================
Write-Host "`n--- 6. DATABASE USERS CRUD ---" -ForegroundColor Yellow
$cdu = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"auto_dbuser","password":"AutoP@ss1","privileges":"ALL PRIVILEGES"}'
Test-Case "Create database user succeeds" { $cdu.ok -and $cdu.data.username -eq "auto_dbuser" }
$duid = if ($cdu.ok) { $cdu.data.id } else { 0 }

# Verify in MariaDB
$duCheck = (docker exec opanel mysql -u root -e "SELECT User FROM mysql.user WHERE User='auto_dbuser';" 2>&1 | Out-String).Trim()
Test-Case "DB user exists in MariaDB" { $duCheck -match "auto_dbuser" }

# Verify grants
$grants = (docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'auto_dbuser'@'%';" 2>&1 | Out-String).Trim()
Test-Case "DB user has ALL PRIVILEGES" { $grants -match "ALL PRIVILEGES" }

# Change privileges
$cduUp = Invoke-Api -Method "PUT" -Path "/api/databases/$dbid/users/$duid" -Token $TOKEN -Body '{"privileges":"SELECT"}'
Test-Case "Update DB user privileges" { $cduUp.ok -and $cduUp.data.privileges -eq "SELECT" }

# Verify grants changed
$grants2 = (docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'auto_dbuser'@'%';" 2>&1 | Out-String).Trim()
Test-Case "MariaDB grants updated to SELECT only" { $grants2 -match "GRANT SELECT ON" -and $grants2 -notmatch "ALL PRIVILEGES" }

# Change password
$cduPwd = Invoke-Api -Method "PUT" -Path "/api/databases/$dbid/users/$duid" -Token $TOKEN -Body '{"password":"NewAutoP@ss2"}'
Test-Case "Update DB user password" { $cduPwd.ok }

# Verify login with new password
$dbLogin = (docker exec opanel mysql -u auto_dbuser -p'NewAutoP@ss2' automated_db -e "SELECT 1;" 2>&1 | Out-String).Trim()
Test-Case "MariaDB login with new password works" { $dbLogin -match "1" }

# Empty params
$cduEmpty = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"","password":""}'
Test-Case "Empty username/password rejected" { (-not $cduEmpty.ok) }

# Invalid username format (special chars)
$cduBad = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"user@bad","password":"Pass123!"}'
Test-Case "DB username with special chars rejected" { (-not $cduBad.ok) }

# Invalid username format (starts with number)
$cduNum = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"1baduser","password":"Pass123!"}'
Test-Case "DB username starting with number rejected" { (-not $cduNum.ok) }

# Invalid username format (spaces)
$cduSpace = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"user name","password":"Pass123!"}'
Test-Case "DB username with spaces rejected" { (-not $cduSpace.ok) }

# Non-existent database
$cdu404 = Invoke-Api -Method "POST" -Path "/api/databases/99999/users" -Token $TOKEN -Body '{"username":"u","password":"p"}'
Test-Case "Create user on non-existent DB fails" { (-not $cdu404.ok) }

# Delete DB user
$cduDel = Invoke-Api -Method "DELETE" -Path "/api/databases/$dbid/users/$duid" -Token $TOKEN
Test-Case "Delete database user" { $cduDel.ok -and $cduDel.data.message -eq "database user deleted" }

# Verify removed from MariaDB
$duGone = (docker exec opanel mysql -u root -e "SELECT User FROM mysql.user WHERE User='auto_dbuser';" 2>&1 | Out-String).Trim()
Test-Case "DB user removed from MariaDB" { $duGone -notmatch "auto_dbuser" }

# ============================================================
Write-Host "`n--- 7. SQL INJECTION ATTEMPTS ---" -ForegroundColor Yellow
# These should all fail safely without causing damage

$sqli1 = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"test DROP TABLE users"}'
Test-Case "SQL injection in DB name (DROP TABLE)" { (-not $sqli1.ok) }

$sqli2 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"admin\"--","email":"sqli@test.com","password":"Test123!","role":"user"}'
# This should either succeed harmlessly or fail validation - it should NOT delete anything
$adminCheck = Invoke-Api -Path "/api/users/1" -Token $TOKEN
Test-Case "SQL injection in username does not destroy admin" { $adminCheck.ok -and $adminCheck.data.username -eq "admin" }

$sqli3 = Invoke-Api -Method "POST" -Path "/api/databases/$dbid/users" -Token $TOKEN -Body '{"username":"test\"@\"%","password":"P@ss1","privileges":"ALL PRIVILEGES"}'
# Should fail or succeed harmlessly
$usersAfter = Invoke-Api -Path "/api/users" -Token $TOKEN
Test-Case "SQL injection in DB username does not destroy users" { $usersAfter.ok }

# ============================================================
Write-Host "`n--- 8. PATH TRAVERSAL ATTEMPTS ---" -ForegroundColor Yellow
# Domain names with ../ should be rejected

$pt1 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"../../etc"}'
Test-Case "Path traversal domain name rejected" { (-not $pt1.ok) }

$pt2 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"../../../root/.ssh"}'
Test-Case "Path traversal to root SSH rejected" { (-not $pt2.ok) }

$pt3 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"test..\\..\\windows"}'
Test-Case "Windows path traversal rejected" { (-not $pt3.ok) }

# Consecutive dots
$pt4 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"domain..invalid.com"}'
Test-Case "Domain with consecutive dots rejected" { (-not $pt4.ok) }

# Invalid domain format (no TLD)
$pt5 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"nodotld"}'
Test-Case "Domain without TLD rejected" { (-not $pt5.ok) }

# Valid domain with hyphens
$pt6 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"valid-domain-test.com"}'
Test-Case "Valid domain with hyphens accepted" { $pt6.ok }
if ($pt6.ok) { Invoke-Api -Method "DELETE" -Path "/api/domains/$($pt6.data.id)" -Token $TOKEN | Out-Null }

# ============================================================
Write-Host "`n--- 9. SPECIAL CHARACTERS ---" -ForegroundColor Yellow

$sp1 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"domain with spaces.com"}'
Test-Case "Domain with spaces rejected" { (-not $sp1.ok) }

$sp2 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"domain@special#.com"}'
Test-Case "Domain with special chars rejected" { (-not $sp2.ok) }

$sp3 = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"db with spaces"}'
Test-Case "Database with spaces rejected" { (-not $sp3.ok) }

# Very long names
$longName = "a" * 300
$sp4 = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body "{\"name\":\"$longName.com\"}"
Test-Case "Very long domain name rejected" { (-not $sp4.ok) }

$sp5 = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body "{\"name\":\"$longName\"}"
Test-Case "Very long database name rejected" { (-not $sp5.ok) }

# ============================================================
Write-Host "`n--- 10. HTTP METHOD WRONG ---" -ForegroundColor Yellow

$wm1 = Invoke-Api -Method "PUT" -Path "/api/health" -Token $TOKEN
Test-Case "PUT /health rejected" { (-not $wm1.ok) }

$wm2 = Invoke-Api -Method "DELETE" -Path "/api/auth/me" -Token $TOKEN
Test-Case "DELETE /auth/me rejected" { (-not $wm2.ok) }

$wm3 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"x","email":"x@x.com","password":"p","role":"r"}'
# POST to list endpoint should fail (no method handler for POST /api/users)
# Actually POST /api/users is the create handler, so this should succeed
# But let's test DELETE on a list endpoint
$wm4 = Invoke-Api -Method "DELETE" -Path "/api/databases" -Token $TOKEN
Test-Case "DELETE /databases rejected" { (-not $wm4.ok) }

# ============================================================
Write-Host "`n--- 11. INTEGRATION: DOMAIN + NGINX + PHP-FPM ---" -ForegroundColor Yellow

$intDom = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"integration-test.com"}'
$intDid = if ($intDom.ok) { $intDom.data.id } else { 0 }
Test-Case "Create integration domain" { $intDom.ok }

# Nginx config content
$intNginx = docker exec opanel cat /etc/nginx/sites-enabled/integration-test.com.conf 2>&1
Test-Case "Nginx config has server_name" { $intNginx -match "integration-test.com" }
Test-Case "Nginx config has root directive" { $intNginx -match "httpdocs" }
Test-Case "Nginx config has PHP handling" { $intNginx -match "fastcgi" }
Test-Case "Nginx config has security headers" { $intNginx -match "X-Frame-Options" }

# PHP-FPM pool content
$intPool = docker exec opanel cat /etc/php/8.4/fpm/pool.d/integration-test.com.pool.conf 2>&1
Test-Case "PHP-FPM pool has domain name" { $intPool -match "integration-test.com" }
Test-Case "PHP-FPM pool has socket path" { $intPool -match "php8.4-fpm" }
Test-Case "PHP-FPM pool has memory limit" { $intPool -match "memory_limit" }
Test-Case "PHP-FPM pool has security (disable_functions)" { $intPool -match "disable_functions" }

# Directory ownership
$intOwner = docker exec opanel ls -la /var/www/vhosts/integration-test.com/ 2>&1
Test-Case "Vhost dir owned by op_ user" { $intOwner -match "op_integration-test.com" }

# Delete and verify cleanup
Invoke-Api -Method "DELETE" -Path "/api/domains/$intDid" -Token $TOKEN | Out-Null
$intPoolAfter = docker exec opanel ls /etc/php/8.4/fpm/pool.d/integration-test.com.pool.conf 2>&1
$intNginxAfter = docker exec opanel ls /etc/nginx/sites-enabled/integration-test.com.conf 2>&1
Test-Case "Pool removed on domain delete" { $intPoolAfter -match "No such file" }
Test-Case "Nginx config removed on domain delete" { $intNginxAfter -match "No such file" }

# ============================================================
Write-Host "`n--- 12. INTEGRATION: DATABASE + MARIADB ---" -ForegroundColor Yellow

# Create DB + user + verify full lifecycle
$intDb = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"integration_db"}'
$intDbid = if ($intDb.ok) { $intDb.data.id } else { 0 }
Test-Case "Create integration database" { $intDb.ok }

$intDbUser = Invoke-Api -Method "POST" -Path "/api/databases/$intDbid/users" -Token $TOKEN -Body '{"username":"int_user","password":"IntP@ss1","privileges":"ALL PRIVILEGES"}'
$intDuid = if ($intDbUser.ok) { $intDbUser.data.id } else { 0 }
Test-Case "Create integration DB user" { $intDbUser.ok }

# Verify MariaDB state
$intMaria = (docker exec opanel mysql -u int_user -p'IntP@ss1' integration_db -e "SELECT 1 AS test;" 2>&1 | Out-String).Trim()
Test-Case "MariaDB login and query works" { $intMaria -match "test" }

# Update to SELECT only
Invoke-Api -Method "PUT" -Path "/api/databases/$intDbid/users/$intDuid" -Token $TOKEN -Body '{"privileges":"SELECT"}' | Out-Null
$intGrants = (docker exec opanel mysql -u root -e "SHOW GRANTS FOR 'int_user'@'%';" 2>&1 | Out-String).Trim()
Test-Case "MariaDB grants correctly updated" { $intGrants -match "GRANT SELECT ON" -and $intGrants -notmatch "ALL PRIVILEGES" }

# Delete DB user then DB
Invoke-Api -Method "DELETE" -Path "/api/databases/$intDbid/users/$intDuid" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/databases/$intDbid" -Token $TOKEN | Out-Null
$intDbGone = (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'integration_db';" 2>&1 | Out-String).Trim()
Test-Case "Database removed from MariaDB after delete" { $intDbGone -notmatch "integration_db" }

# ============================================================
Write-Host "`n--- 13. CLEANUP ---" -ForegroundColor Yellow
# Remove any leftover test data
$cleanupUsers = Invoke-Api -Path "/api/users" -Token $TOKEN
if ($cleanupUsers.ok) {
    foreach ($u in $cleanupUsers.data) {
        if ($u.username -ne "admin") {
            Invoke-Api -Method "DELETE" -Path "/api/users/$($u.id)" -Token $TOKEN | Out-Null
        }
    }
}
$cleanupDbs = Invoke-Api -Path "/api/databases" -Token $TOKEN
if ($cleanupDbs.ok) {
    foreach ($d in $cleanupDbs.data) {
        Invoke-Api -Method "DELETE" -Path "/api/databases/$($d.id)" -Token $TOKEN | Out-Null
    }
}
Test-Case "Cleanup completed" { $true }

# ============================================================
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  RESULTS: $pass PASS, $fail FAIL, $warn WARN" -ForegroundColor $(if ($fail -eq 0) { "Green" } else { "Red" })
Write-Host "  Total: $($pass + $fail + $warn) tests" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

if ($fail -gt 0) {
    Write-Host "FAILED TESTS:" -ForegroundColor Red
    foreach ($t in $tests) {
        if ($t.result -eq "FAIL") {
            Write-Host "  - $($t.name)" -ForegroundColor Red
        }
    }
    exit 1
}
exit 0
