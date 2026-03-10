param(
    [switch]$QuietLogs
)

$ErrorActionPreference = "Stop"

# Always run from repository root.
Set-Location (Resolve-Path "$PSScriptRoot\..")

function Stop-ProcessOnPort {
    param(
        [Parameter(Mandatory = $true)]
        [int]$Port
    )

    $procIds = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -Unique

    foreach ($procId in $procIds) {
        if ($procId -and $procId -ne $PID) {
            try {
                $proc = Get-Process -Id $procId -ErrorAction Stop
                Write-Host "[STARTUP] Port ${Port} is occupied by $($proc.ProcessName) (PID: $procId). Stopping it..."
                Stop-Process -Id $procId -Force -ErrorAction Stop
            } catch {
                Write-Host "[STARTUP] Failed to stop process on port ${Port} (PID: $procId): $($_.Exception.Message)"
            }
        }
    }
}

Stop-ProcessOnPort -Port 8080

$preferredGo = "D:\go\bin"
if (Test-Path "$preferredGo\go.exe") {
    if (-not (($env:Path -split ";") -contains $preferredGo)) {
        $env:Path = "$preferredGo;$env:Path"
    }
} elseif (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go not found. Please install Go or add go.exe to PATH."
    exit 1
}

if ([string]::IsNullOrWhiteSpace($env:JWT_SECRET)) {
    $bytes = New-Object byte[] 32
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $env:JWT_SECRET = ([System.BitConverter]::ToString($bytes)).Replace("-", "").ToLower()
    Write-Host "[STARTUP] JWT_SECRET not set. Generated an in-memory secret for this run."
} else {
    Write-Host "[STARTUP] JWT_SECRET already set in environment."
}

$goVersion = (& go version)
Write-Host "[STARTUP] Using $goVersion"
Write-Host "[STARTUP] Starting WSwriter on http://127.0.0.1:8080"

if (-not $QuietLogs) {
    & go run WSwriter.go
    exit $LASTEXITCODE
}

# Only suppress known non-critical noise lines in quiet mode.
$noisePatterns = @(
    "\[WARN\] Hugo server URL not detected",
    "\[AUDIT\] HTTPS not configured",
    "\[HUGO-WARN\] WARN\s+Widget links not found"
)

& go run WSwriter.go 2>&1 | ForEach-Object {
    $line = $_.ToString()
    $isNoise = $false

    foreach ($pattern in $noisePatterns) {
        if ($line -match $pattern) {
            $isNoise = $true
            break
        }
    }

    if (-not $isNoise) {
        Write-Host $line
    }
}

exit $LASTEXITCODE
