# run_services.ps1 - one-command build & start all SLG services (dev env)
#
# Usage:
#   pwsh -File scripts/run_services.ps1             # default env=dev instance=1
#   pwsh -File scripts/run_services.ps1 -Instance 2
#
# Prereq: infra (MySQL/Redis/etcd) running (envs/start.ps1); go in PATH
# Stop:   pwsh -File scripts/stop_services.ps1

param(
    [string]$Instance = "1",
    [string]$Env = "dev"
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Bin = Join-Path $Root "bin"
New-Item -ItemType Directory -Force $Bin | Out-Null

$services = @("login", "gateway", "battle", "battle_record", "worldmap", "game")

Write-Host "[build] compiling service binaries -> $Bin" -ForegroundColor Cyan
foreach ($svc in $services) {
    Write-Host "  build $svc"
    Push-Location $Root
    go build -o (Join-Path $Bin "$svc.exe") ./services/$svc
    if ($LASTEXITCODE -ne 0) { throw "build $svc failed" }
    Pop-Location
}

Write-Host "[start] starting services (env=$Env, instance=$Instance)" -ForegroundColor Cyan
foreach ($svc in $services) {
    $exe = Join-Path $Bin "$svc.exe"
    $args = @("-env", $Env)
    # game/worldmap/battle/battle_record pair by instance; login/gateway shared (default)
    if ($svc -in @("game", "worldmap", "battle", "battle_record")) {
        $args += @("-instance", $Instance)
    }
    $log = Join-Path $Bin "$svc.log"
    $p = Start-Process -FilePath $exe -ArgumentList $args -WorkingDirectory $Root `
        -RedirectStandardOutput $log -RedirectStandardError "$log.err" `
        -WindowStyle Hidden -PassThru
    Write-Host "  $svc -> PID $($p.Id) (log: $log)"
}

Write-Host "[health] waiting for service ports" -ForegroundColor Cyan
$ports = @{
    "login" = 14001; "gateway" = 13001; "game" = 11001
    "worldmap" = 50060; "battle" = 12001; "battle_record" = 12002
}
foreach ($svc in $services) {
    $port = $ports[$svc]
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
        if (Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue) {
            Write-Host "  OK  $svc :$port"
            $ready = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $ready) { Write-Host "  WARN $svc :$port not ready (see $Bin\$svc.log)" -ForegroundColor Yellow }
}

Write-Host ""
Write-Host "All services started. Client demo: go run ./tools/slg_client" -ForegroundColor Green
