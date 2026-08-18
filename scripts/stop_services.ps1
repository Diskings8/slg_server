# stop_services.ps1 — 停止全部 SLG 服务进程
#
# 用法: pwsh -File scripts/stop_services.ps1

$names = @("login", "gateway", "game", "worldmap", "battle", "battle_record")
foreach ($name in $names) {
    $procs = Get-Process -Name $name -ErrorAction SilentlyContinue
    if ($procs) {
        $procs | Stop-Process -Force
        Write-Host "stopped $name ($($procs.Count) process)"
    } else {
        Write-Host "skip   $name (not running)"
    }
}
