$ErrorActionPreference = "Stop"

function Stop-ProcessOnPort {
    param(
        [Parameter(Mandatory = $true)]
        [int]$Port
    )

    $procIds = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -Unique

    if (-not $procIds) {
        Write-Host "[STOP] No listening process found on port ${Port}."
        return
    }

    foreach ($procId in $procIds) {
        try {
            $proc = Get-Process -Id $procId -ErrorAction Stop
            Write-Host "[STOP] Stopping $($proc.ProcessName) (PID: $procId) on port ${Port}..."
            Stop-Process -Id $procId -Force -ErrorAction Stop
        } catch {
            Write-Host "[STOP] Failed to stop PID $procId on port ${Port}: $($_.Exception.Message)"
        }
    }
}

Stop-ProcessOnPort -Port 8080
