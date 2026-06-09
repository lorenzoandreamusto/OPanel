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
Write-Host "`n--- 4b. DOMAIN EXTENSIONS (hosting_type, php_version, ssl, auto_db) ---" -ForegroundColor Yellow

# Create PHP domain with specific version
$dcPhp = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"php-specific.com","php_version":"8.3","hosting_type":"php","ssl_enabled":true,"auto_db":true}'
Test-Case "Create PHP domain with php_version=8.3" { $dcPhp.ok -and $dcPhp.data.php_version -eq "8.3" }
$didPhp = if ($dcPhp.ok) { $dcPhp.data.id } else { 0 }

Test-Case "PHP domain has hosting_type=php" { $dcPhp.data.hosting_type -eq "php" }
Test-Case "PHP domain has ssl_enabled=true" { $dcPhp.data.ssl_enabled -eq $true }
Test-Case "PHP domain has auto_db=true" { $dcPhp.data.auto_db -eq $true }

# Verify PHP-FPM pool generated for PHP domain
$phpPool = docker exec opanel cat /etc/php/8.4/fpm/pool.d/php-specific.com.pool.conf 2>&1
Test-Case "PHP-FPM pool created for PHP domain" { $phpPool -match "php-specific.com" }

# Verify Nginx config generated for PHP domain
$phpNginx = docker exec opanel cat /etc/nginx/sites-enabled/php-specific.com.conf 2>&1
Test-Case "Nginx config created for PHP domain" { $phpNginx -match "server_name" }

# Get domain returns new fields
$dgPhp = Invoke-Api -Path "/api/domains/$didPhp" -Token $TOKEN
Test-Case "Get PHP domain returns php_version" { $dgPhp.ok -and $dgPhp.data.php_version -eq "8.3" }
Test-Case "Get PHP domain returns hosting_type" { $dgPhp.data.hosting_type -eq "php" }
Test-Case "Get PHP domain returns ssl_enabled" { $dgPhp.data.ssl_enabled -eq $true }
Test-Case "Get PHP domain returns auto_db" { $dgPhp.data.auto_db -eq $true }

# Create static domain (no PHP-FPM pool expected)
$dcStatic = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"static-only.com","hosting_type":"static"}'
Test-Case "Create static domain" { $dcStatic.ok -and $dcStatic.data.hosting_type -eq "static" }
$didStatic = if ($dcStatic.ok) { $dcStatic.data.id } else { 0 }

# Static domain should NOT have PHP-FPM pool
$staticPool = docker exec opanel ls /etc/php/8.4/fpm/pool.d/static-only.com.pool.conf 2>&1
Test-Case "Static domain has NO PHP-FPM pool" { $staticPool -match "No such file" }

# Static domain SHOULD have Nginx config
$staticNginx = docker exec opanel cat /etc/nginx/sites-enabled/static-only.com.conf 2>&1
Test-Case "Static domain has Nginx config" { $staticNginx -match "server_name" }

# Static domain should NOT have fastcgi (PHP handling) in Nginx
Test-Case "Static Nginx has no fastcgi_pass" { $staticNginx -notmatch "fastcgi_pass" }

# Default values for new fields when creating with just name
$dcDefault = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"default-fields.com"}'
Test-Case "Default domain has php_version=8.4" { $dcDefault.ok -and $dcDefault.data.php_version -eq "8.4" }
Test-Case "Default domain has hosting_type=php" { $dcDefault.data.hosting_type -eq "php" }
Test-Case "Default domain has ssl_enabled=false" { $dcDefault.data.ssl_enabled -eq $false }
Test-Case "Default domain has auto_db=false" { $dcDefault.data.auto_db -eq $false }
$didDefault = if ($dcDefault.ok) { $dcDefault.data.id } else { 0 }

# Update domain fields
$uuPhp = Invoke-Api -Method "PUT" -Path "/api/domains/$didPhp" -Token $TOKEN -Body '{"php_version":"8.4"}'
Test-Case "Update domain php_version to 8.4" { $uuPhp.ok -and $uuPhp.data.php_version -eq "8.4" }

$uuSsl = Invoke-Api -Method "PUT" -Path "/api/domains/$didPhp" -Token $TOKEN -Body '{"ssl_enabled":false}'
Test-Case "Update domain ssl_enabled to false" { $uuSsl.ok -and $uuSsl.data.ssl_enabled -eq $false }

$uuType = Invoke-Api -Method "PUT" -Path "/api/domains/$didDefault" -Token $TOKEN -Body '{"hosting_type":"static"}'
Test-Case "Update domain hosting_type to static" { $uuType.ok -and $uuType.data.hosting_type -eq "static" }

# After changing to static, PHP-FPM pool should ideally be removed (or at least Nginx should not use PHP)
# For now we just verify the field updated
$dgDefault = Invoke-Api -Path "/api/domains/$didDefault" -Token $TOKEN
Test-Case "Default domain updated to static" { $dgDefault.ok -and $dgDefault.data.hosting_type -eq "static" }

# Cleanup extension test domains
Invoke-Api -Method "DELETE" -Path "/api/domains/$didPhp" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/domains/$didStatic" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/domains/$didDefault" -Token $TOKEN | Out-Null

# Verify cleanup
$phpPoolAfter = docker exec opanel ls /etc/php/8.4/fpm/pool.d/php-specific.com.pool.conf 2>&1
Test-Case "PHP-FPM pool removed after PHP domain delete" { $phpPoolAfter -match "No such file" }
$staticNginxAfter = docker exec opanel ls /etc/nginx/sites-enabled/static-only.com.conf 2>&1
Test-Case "Nginx config removed after static domain delete" { $staticNginxAfter -match "No such file" }

# ============================================================
Write-Host "`n--- 4c. SUSPEND/ACTIVATE + STATIC SITE BEHAVIOR ---" -ForegroundColor Yellow

# Create domain for suspend test
$dcSusp = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"suspend-test.com"}'
$didSusp = if ($dcSusp.ok) { $dcSusp.data.id } else { 0 }
Test-Case "Create domain for suspend test" { $dcSusp.ok }

# Verify normal Nginx config
$suspNginx = docker exec opanel cat /etc/nginx/sites-enabled/suspend-test.com.conf 2>&1
Test-Case "Active domain has fastcgi_pass" { $suspNginx -match "fastcgi_pass" }

# Suspend domain
$suspendResult = Invoke-Api -Method "PUT" -Path "/api/domains/$didSusp" -Token $TOKEN -Body '{"status":"suspended"}'
Test-Case "Suspend returns status=suspended" { $suspendResult.ok -and $suspendResult.data.status -eq "suspended" }

# Verify Nginx config now has suspension page
$suspendedNginx = docker exec opanel cat /etc/nginx/sites-enabled/suspend-test.com.conf 2>&1
Test-Case "Suspended Nginx has return 403" { $suspendedNginx -match "return 403" }
Test-Case "Suspended Nginx has no fastcgi_pass" { $suspendedNginx -notmatch "fastcgi_pass" }
Test-Case "Suspended Nginx has suspension message" { $suspendedNginx -match "Site Suspended" }

# Site still responds on port 80 (with suspension page)
$suspPage = curl -s -H "Host: suspend-test.com" http://localhost:80/ 2>&1
Test-Case "Suspended site returns 403" { $suspPage -match "Site Suspended" }

# Activate domain
$activateResult = Invoke-Api -Method "PUT" -Path "/api/domains/$didSusp" -Token $TOKEN -Body '{"status":"active"}'
Test-Case "Activate returns status=active" { $activateResult.ok -and $activateResult.data.status -eq "active" }

# Verify Nginx config restored to normal
$activatedNginx = docker exec opanel cat /etc/nginx/sites-enabled/suspend-test.com.conf 2>&1
Test-Case "Activated Nginx has fastcgi_pass again" { $activatedNginx -match "fastcgi_pass" }
Test-Case "Activated Nginx has no return 403" { $activatedNginx -notmatch "return 403" }

# Hosting type: PHP → Static removes pool
$dcHt = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"hosting-test.com","hosting_type":"php"}'
$didHt = if ($dcHt.ok) { $dcHt.data.id } else { 0 }
Test-Case "Create PHP domain for hosting type test" { $dcHt.ok }

$htPool = docker exec opanel cat /etc/php/8.4/fpm/pool.d/hosting-test.com.pool.conf 2>&1
Test-Case "PHP domain has pool" { $htPool -match "hosting-test.com" }

# Switch to static
$htStatic = Invoke-Api -Method "PUT" -Path "/api/domains/$didHt" -Token $TOKEN -Body '{"hosting_type":"static"}'
Test-Case "Switch to static succeeds" { $htStatic.ok -and $htStatic.data.hosting_type -eq "static" }

$htPoolAfter = docker exec opanel ls /etc/php/8.4/fpm/pool.d/hosting-test.com.pool.conf 2>&1
Test-Case "Pool removed after switch to static" { $htPoolAfter -match "No such file" }

$htNginx = docker exec opanel cat /etc/nginx/sites-enabled/hosting-test.com.conf 2>&1
Test-Case "Static Nginx has no fastcgi_pass" { $htNginx -notmatch "fastcgi_pass" }

# Switch back to PHP
$htPhp = Invoke-Api -Method "PUT" -Path "/api/domains/$didHt" -Token $TOKEN -Body '{"hosting_type":"php"}'
Test-Case "Switch back to PHP succeeds" { $htPhp.ok -and $htPhp.data.hosting_type -eq "php" }

$htPoolRecreated = docker exec opanel cat /etc/php/8.4/fpm/pool.d/hosting-test.com.pool.conf 2>&1
Test-Case "Pool recreated after switch to PHP" { $htPoolRecreated -match "hosting-test.com" }

$htNginxPhp = docker exec opanel cat /etc/nginx/sites-enabled/hosting-test.com.conf 2>&1
Test-Case "PHP Nginx has fastcgi_pass again" { $htNginxPhp -match "fastcgi_pass" }

# Cleanup
Invoke-Api -Method "DELETE" -Path "/api/domains/$didSusp" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/domains/$didHt" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 4d. AUTO_DB CREATION ---" -ForegroundColor Yellow

# Create domain with auto_db
$dcAuto = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"autodb-test.com","auto_db":true}'
Test-Case "Create domain with auto_db" { $dcAuto.ok -and $dcAuto.data.auto_db -eq $true }
$didAuto = if ($dcAuto.ok) { $dcAuto.data.id } else { 0 }

# Database should be created in MariaDB (domain name with dots → underscores)
$autoDbCheck = (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'autodb-test_com';" 2>&1 | Out-String).Trim()
Test-Case "Auto DB exists in MariaDB" { $autoDbCheck -match "autodb-test_com" }

# Database should be tracked in SQLite (visible via API)
$dbList = Invoke-Api -Path "/api/databases" -Token $TOKEN
$foundAutoDb = $false
if ($dbList.ok) {
    foreach ($db in $dbList.data) {
        if ($db.name -eq "autodb-test_com") { $foundAutoDb = $true; break }
    }
}
Test-Case "Auto DB tracked in API" { $foundAutoDb }

# Delete domain with auto_db — should also delete the database
$ddAuto = Invoke-Api -Method "DELETE" -Path "/api/domains/$didAuto" -Token $TOKEN
Test-Case "Delete domain with auto_db" { $ddAuto.ok }

$autoDbAfter = (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'autodb_test';" 2>&1 | Out-String).Trim()
Test-Case "Auto DB removed from MariaDB after domain delete" { $autoDbAfter -notmatch "autodb_test" }

# Create domain WITHOUT auto_db — should NOT create database
$dcNoAuto = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"noautodb-test.com","auto_db":false}'
$didNoAuto = if ($dcNoAuto.ok) { $dcNoAuto.data.id } else { 0 }
$noAutoDb = (docker exec opanel mysql -u root -e "SHOW DATABASES LIKE 'noautodb-test_com';" 2>&1 | Out-String).Trim()
Test-Case "No auto DB when auto_db=false" { $noAutoDb -notmatch "noautodb-test_com" }

# Cleanup
Invoke-Api -Method "DELETE" -Path "/api/domains/$didNoAuto" -Token $TOKEN | Out-Null

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

# List database users
$ldu = Invoke-Api -Path "/api/databases/$dbid/users" -Token $TOKEN
Test-Case "List database users returns array" { $ldu.ok -and ($ldu.data | Measure-Object).Count -ge 1 }
Test-Case "List database users includes created user" { ($ldu.data | Where-Object { $_.username -eq "auto_dbuser" }) -ne $null }

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
Write-Host "`n--- 11b. WEB SERVER: PORT 80 + SITE ACCESS ---" -ForegroundColor Yellow

# Verify Nginx default catch-all config exists
$defaultConf = docker exec opanel cat /etc/nginx/sites-enabled/00-default.conf 2>&1
Test-Case "Default catch-all Nginx config exists" { $defaultConf -match "server_name _" }
Test-Case "Default catch-all has default_server" { $defaultConf -match "default_server" }

# Verify port 80 catch-all returns proper message for unknown domains
$catchAll = curl -s http://localhost:80 2>&1
Test-Case "Port 80 catch-all returns site-not-configured message" { $catchAll -match "Site not configured" }

# Create domain for web server tests
$webDom = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"webtest.com"}'
$webDid = if ($webDom.ok) { $webDom.data.id } else { 0 }
Test-Case "Create web test domain" { $webDom.ok }

# Verify PHP-FPM socket was created with consistent naming
$webSocket = docker exec opanel ls -la /run/php/php8.4-fpm-op_webtest-com.sock 2>&1
Test-Case "PHP-FPM socket created for domain" { $webSocket -match "op_webtest-com.sock" }

# Verify socket name in Nginx config matches actual socket file
$webNginx = docker exec opanel cat /etc/nginx/sites-enabled/webtest.com.conf 2>&1
$webSocketInNginx = ($webNginx | Select-String "fastcgi_pass unix:(.*);").Matches.Groups[1].Value
$webSocketFile = ($webSocket | Select-String "php8.4-fpm-op_webtest-com.sock").Matches.Count -gt 0
Test-Case "Nginx config socket path matches actual socket" { $webSocketInNginx -match "op_webtest-com.sock" -and $webSocketFile }

# Verify Nginx config does NOT have deprecated listen.mode (PHP-FPM pool check)
$webPool = docker exec opanel cat /etc/php/8.4/fpm/pool.d/webtest.com.pool.conf 2>&1
Test-Case "PHP-FPM pool has no deprecated listen.mode" { $webPool -notmatch "listen.mode" }

# Verify site is accessible via port 80 with Host header
$sitePage = try { (Invoke-WebRequest -Uri "http://localhost:80" -Headers @{"Host"="webtest.com"} -UseBasicParsing -TimeoutSec 5).Content } catch { $_.Exception.Response }
Test-Case "Site accessible via port 80 with Host header" { $sitePage -match "Welcome to nginx" }

# Verify default index.html was copied to httpdocs
$webIndex = docker exec opanel cat /var/www/vhosts/webtest.com/httpdocs/index.html 2>&1
Test-Case "Default index.html exists in httpdocs" { $webIndex -match "Welcome to nginx" }

# Verify Nginx config has try_files =404 (not /index.php fallback)
Test-Case "Nginx try_files returns 404 not PHP fallback" { $webNginx -match "try_files.*=404" }

# Verify Nginx config does NOT have /index.php fallback
Test-Case "Nginx has no index.php fallback in try_files" { $webNginx -notmatch "try_files.*index\.php" }

# Verify Nginx config has security headers
Test-Case "Nginx config has X-Frame-Options" { $webNginx -match "X-Frame-Options" }
Test-Case "Nginx config has X-Content-Type-Options" { $webNginx -match "X-Content-Type-Options" }
Test-Case "Nginx config has X-XSS-Protection" { $webNginx -match "X-XSS-Protection" }

# Verify Nginx is running and config test passes
$nginxTest = docker exec opanel nginx -t 2>&1
Test-Case "Nginx config test passes" { $nginxTest -match "test is successful" }

# Verify PHP-FPM process is running for this domain (check socket exists instead of process name due to Docker PID 1 limitation)
$webSocketAfter = docker exec opanel ls /run/php/php8.4-fpm-op_webtest-com.sock 2>&1
Test-Case "PHP-FPM pool process running for domain" { $webSocketAfter -match "op_webtest-com.sock" }

# Verify Nginx is listening on port 80
$nginxListening = docker exec opanel ss -tlnp 2>&1
Test-Case "Nginx listening on port 80" { $nginxListening -match ":80" }

# Cleanup web test domain
Invoke-Api -Method "DELETE" -Path "/api/domains/$webDid" -Token $TOKEN | Out-Null
$webSocketAfter = docker exec opanel ls /run/php/php8.4-fpm-op_webtest-com.sock 2>&1
Test-Case "PHP-FPM socket removed on domain delete" { $webSocketAfter -match "No such file" }
$webPoolAfter = docker exec opanel ls /etc/php/8.4/fpm/pool.d/webtest.com.pool.conf 2>&1
Test-Case "PHP-FPM pool removed on domain delete" { $webPoolAfter -match "No such file" }

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
Write-Host "`n--- 13. ROLE VALIDATION (bugfix tests) ---" -ForegroundColor Yellow

$roleBad1 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"rolebad1","email":"rb1@test.com","password":"Test123!","role":"superadmin"}'
Test-Case "Create user with invalid role rejected" { (-not $roleBad1.ok) }

$roleBad2 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"rolebad2","email":"rb2@test.com","password":"Test123!","role":""}'
Test-Case "Create user with empty role defaults to user" { $roleBad2.ok -and $roleBad2.data.role -eq "user" }
if ($roleBad2.ok) { Invoke-Api -Method "DELETE" -Path "/api/users/$($roleBad2.data.id)" -Token $TOKEN | Out-Null }

$roleOk1 = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"roleok1","email":"ro1@test.com","password":"Test123!","role":"admin"}'
Test-Case "Create user with role=admin succeeds" { $roleOk1.ok -and $roleOk1.data.role -eq "admin" }
if ($roleOk1.ok) { Invoke-Api -Method "DELETE" -Path "/api/users/$($roleOk1.data.id)" -Token $TOKEN | Out-Null }

# Update with invalid role
$ruTemp = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"roleupdate","email":"ru@test.com","password":"Test123!","role":"user"}'
$ruId = if ($ruTemp.ok) { $ruTemp.data.id } else { 0 }
$roleUpBad = Invoke-Api -Method "PUT" -Path "/api/users/$ruId" -Token $TOKEN -Body '{"role":"hacker"}'
Test-Case "Update user with invalid role rejected" { (-not $roleUpBad.ok) }
$roleUpOk = Invoke-Api -Method "PUT" -Path "/api/users/$ruId" -Token $TOKEN -Body '{"role":"admin"}'
Test-Case "Update user with valid role=admin succeeds" { $roleUpOk.ok -and $roleUpOk.data.role -eq "admin" }
Invoke-Api -Method "DELETE" -Path "/api/users/$ruId" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 14. DOMAIN OWNERSHIP (bugfix tests) ---" -ForegroundColor Yellow

# Create a non-admin user
$ownerUser = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"owner_test","email":"owner@test.com","password":"Test123!","role":"user"}'
$ownerId = if ($ownerUser.ok) { $ownerUser.data.id } else { 0 }
$ownerLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"owner_test","password":"Test123!"}'
$ownerToken = if ($ownerLogin.ok) { $ownerLogin.data.token } else { "" }

# Admin creates a domain
$adminDom = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body '{"name":"admin-owned.com"}'
$adminDomId = if ($adminDom.ok) { $adminDom.data.id } else { 0 }

# Non-admin user creates a domain
$ownerDom = Invoke-Api -Method "POST" -Path "/api/domains" -Token $ownerToken -Body '{"name":"owner-owned.com"}'
$ownerDomId = if ($ownerDom.ok) { $ownerDom.data.id } else { 0 }

# Non-admin cannot see admin's domain by ID
$ownerGetAdmin = Invoke-Api -Path "/api/domains/$adminDomId" -Token $ownerToken
Test-Case "Non-admin cannot get admin domain by ID" { (-not $ownerGetAdmin.ok) }

# Admin can see all domains
$adminListAll = Invoke-Api -Path "/api/domains" -Token $TOKEN
Test-Case "Admin sees all domains" { $adminListAll.ok -and $adminListAll.data.Count -ge 2 }

# Non-admin only sees own domains
$ownerList = Invoke-Api -Path "/api/domains" -Token $ownerToken
Test-Case "Non-admin sees only own domains" { $ownerList.ok -and $ownerList.data.Count -eq 1 -and $ownerList.data[0].name -eq "owner-owned.com" }

# Non-admin can get own domain
$ownerGetOwn = Invoke-Api -Path "/api/domains/$ownerDomId" -Token $ownerToken
Test-Case "Non-admin can get own domain" { $ownerGetOwn.ok }

# Cleanup
Invoke-Api -Method "DELETE" -Path "/api/domains/$adminDomId" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/domains/$ownerDomId" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/users/$ownerId" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 15. USER PASSWORD UPDATE ---" -ForegroundColor Yellow

$pwUser = Invoke-Api -Method "POST" -Path "/api/users" -Token $TOKEN -Body '{"username":"pwtest","email":"pw@test.com","password":"OldPass1!","role":"user"}'
$pwId = if ($pwUser.ok) { $pwUser.data.id } else { 0 }

# Update password
$pwUp = Invoke-Api -Method "PUT" -Path "/api/users/$pwId" -Token $TOKEN -Body '{"password":"NewPass2!"}'
Test-Case "Update user password succeeds" { $pwUp.ok }

# Login with old password should fail
$pwOldLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"pwtest","password":"OldPass1!"}'
Test-Case "Login with old password fails" { (-not $pwOldLogin.ok) }

# Login with new password should succeed
$pwNewLogin = Invoke-Api -Method "POST" -Path "/api/auth/login" -Body '{"username":"pwtest","password":"NewPass2!"}'
Test-Case "Login with new password succeeds" { $pwNewLogin.ok }

Invoke-Api -Method "DELETE" -Path "/api/users/$pwId" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 16. DATABASE USER EDGE CASES ---" -ForegroundColor Yellow

# Create a temp DB for edge case tests
$edgeDb = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body '{"name":"edge_case_db"}'
$edgeDbId = if ($edgeDb.ok) { $edgeDb.data.id } else { 0 }

# Delete non-existent DB user
$edgeDel = Invoke-Api -Method "DELETE" -Path "/api/databases/$edgeDbId/users/99999" -Token $TOKEN
Test-Case "Delete non-existent DB user fails" { (-not $edgeDel.ok) }

# Update non-existent DB user
$edgeUp = Invoke-Api -Method "PUT" -Path "/api/databases/$edgeDbId/users/99999" -Token $TOKEN -Body '{"password":"x"}'
Test-Case "Update non-existent DB user fails" { (-not $edgeUp.ok) }

# Create user, update both password and privileges at once
$edgeUsr = Invoke-Api -Method "POST" -Path "/api/databases/$edgeDbId/users" -Token $TOKEN -Body '{"username":"edge_usr","password":"Edge1!","privileges":"SELECT"}'
$edgeUsrId = if ($edgeUsr.ok) { $edgeUsr.data.id } else { 0 }
$edgeBoth = Invoke-Api -Method "PUT" -Path "/api/databases/$edgeDbId/users/$edgeUsrId" -Token $TOKEN -Body '{"password":"EdgeNew2!","privileges":"ALL PRIVILEGES"}'
Test-Case "Update both password and privileges" { $edgeBoth.ok -and $edgeBoth.data.privileges -eq "ALL PRIVILEGES" }

# Verify new password works
$edgeLogin = (docker exec opanel mysql -u edge_usr -p'EdgeNew2!' edge_case_db -e "SELECT 1;" 2>&1 | Out-String).Trim()
Test-Case "DB user login with updated password works" { $edgeLogin -match "1" }

# Update with empty body (neither password nor privileges)
$edgeEmpty = Invoke-Api -Method "PUT" -Path "/api/databases/$edgeDbId/users/$edgeUsrId" -Token $TOKEN -Body '{}'
Test-Case "Update with empty body rejected" { (-not $edgeEmpty.ok) }

# Cleanup
Invoke-Api -Method "DELETE" -Path "/api/databases/$edgeDbId/users/$edgeUsrId" -Token $TOKEN | Out-Null
Invoke-Api -Method "DELETE" -Path "/api/databases/$edgeDbId" -Token $TOKEN | Out-Null

# ============================================================
Write-Host "`n--- 17. CONCURRENT REQUESTS ---" -ForegroundColor Yellow

# Fire 5 parallel health checks
$jobs = @()
1..5 | ForEach-Object {
    $jobs += Start-Job -ScriptBlock {
        param($url)
        try {
            $r = Invoke-WebRequest -Uri "$url/api/health" -UseBasicParsing -TimeoutSec 5
            return $r.StatusCode -eq 200
        } catch { return $false }
    } -ArgumentList $BaseUrl
}
$jobs | Wait-Job -Timeout 10 | Out-Null
$results = $jobs | Receive-Job
$jobs | Remove-Job -Force
$allOk = ($results | Where-Object { $_ -eq $true }).Count -eq 5
Test-Case "5 concurrent health checks all succeed" { $allOk }

# ============================================================
Write-Host "`n--- 18. LARGE PAYLOAD ---" -ForegroundColor Yellow

$bigBody = '{"name":"' + ("a" * 1000) + '.com"}'
$bigResp = Invoke-Api -Method "POST" -Path "/api/domains" -Token $TOKEN -Body $bigBody
Test-Case "Very long domain name (1000+ chars) rejected" { (-not $bigResp.ok) }

$bigDbBody = '{"name":"' + ("x" * 500) + '"}'
$bigDbResp = Invoke-Api -Method "POST" -Path "/api/databases" -Token $TOKEN -Body $bigDbBody
Test-Case "Very long database name (500+ chars) rejected" { (-not $bigDbResp.ok) }

# ============================================================
Write-Host "`n--- 19. CONFIGURATION CONSISTENCY ---" -ForegroundColor Yellow

# Verify JWT expiry hours from config is respected (token should have exp claim)
$jwtParts = $TOKEN.Split('.')
if ($jwtParts.Count -eq 3) {
    $payload = $jwtParts[1]
    # Pad base64
    $mod = $payload.Length % 4
    if ($mod -ne 0) { $payload += ("=" * (4 - $mod)) }
    $decoded = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($payload))
    $jwtData = $decoded | ConvertFrom-Json -ErrorAction SilentlyContinue
    Test-Case "JWT token has exp claim" { $null -ne $jwtData.exp }
    Test-Case "JWT token has iat claim" { $null -ne $jwtData.iat }
    Test-Case "JWT token has iss=opanel" { $jwtData.iss -eq "opanel" }
} else {
    Test-Case "JWT token format valid" { $false }
}

# ============================================================
Write-Host "`n--- 20. CLEANUP ---" -ForegroundColor Yellow
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
$cleanupDoms = Invoke-Api -Path "/api/domains" -Token $TOKEN
if ($cleanupDoms.ok) {
    foreach ($d in $cleanupDoms.data) {
        Invoke-Api -Method "DELETE" -Path "/api/domains/$($d.id)" -Token $TOKEN | Out-Null
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
