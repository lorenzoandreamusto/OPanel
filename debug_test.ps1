try {
    Invoke-WebRequest -Uri 'http://localhost:8443/api/auth/login' -Method POST -Body ([System.Text.Encoding]::UTF8.GetBytes('{"username":"admin","password":"wrong"}')) -ContentType 'application/json' -UseBasicParsing -ErrorAction Stop
} catch {
    if ($_.ErrorDetails) {
        Write-Host "ErrorDetails type: $($_.ErrorDetails.GetType().FullName)"
        Write-Host "ErrorDetails.Message: [$($_.ErrorDetails.Message)]"
        Write-Host "ErrorDetails.Message length: $($_.ErrorDetails.Message.Length)"
        $raw = $_.ErrorDetails.RawMessage
        if ($raw) { Write-Host "ErrorDetails.RawMessage: [$raw]" }
    }
    if ($_.Exception.Response) {
        $code = [int]$_.Exception.Response.StatusCode
        Write-Host "Status: $code"
        $stream = $_.Exception.Response.GetResponseStream()
        Write-Host "Stream null: $($stream -eq $null)"
        if ($stream) {
            Write-Host "Stream canread: $($stream.CanRead)"
            Write-Host "Stream length: $($stream.Length)"
            $buf = New-Object byte[] 1024
            $n = $stream.Read($buf, 0, 1024)
            Write-Host "Stream read $n bytes"
            if ($n -gt 0) { Write-Host "Data: $([System.Text.Encoding]::UTF8.GetString($buf, 0, $n))" }
        }
    }
    Write-Host "Exception type: $($_.Exception.GetType().FullName)"
    Write-Host "Inner: $($_.Exception.InnerException)"
}
