# 启动所有基础服务（MySQL, Redis, etcd）
Write-Host "Starting services (MySQL, Redis, etcd)..." -ForegroundColor Cyan
docker compose up -d

Write-Host "`nService status:" -ForegroundColor Cyan
docker compose ps
