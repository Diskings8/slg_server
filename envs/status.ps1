# 查看所有基础服务运行状态
Write-Host "=== docker compose ps ===" -ForegroundColor Cyan
docker compose ps

Write-Host "`n=== 各服务健康检查 ===" -ForegroundColor Cyan
Write-Host "MySQL:" -NoNewline
docker compose exec mysql mysqladmin ping -h localhost -u root -proot123456 --silent 2>$null
if ($LASTEXITCODE -eq 0) { Write-Host "  healthy" -ForegroundColor Green } else { Write-Host "  unhealthy" -ForegroundColor Red }

Write-Host "Redis:" -NoNewline
$redisPong = docker compose exec redis redis-cli ping 2>$null
if ($redisPong -eq "PONG") { Write-Host "  healthy" -ForegroundColor Green } else { Write-Host "  unhealthy" -ForegroundColor Red }

Write-Host "etcd:" -NoNewline
$etcdHealth = docker compose exec etcd etcdctl endpoint health 2>&1
if ($LASTEXITCODE -eq 0) { Write-Host "  healthy" -ForegroundColor Green } else { Write-Host "  unhealthy" -ForegroundColor Red }
