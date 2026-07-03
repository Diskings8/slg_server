$src = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol\src")
$out = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol")
$module = "server.slg.com/api/protocol"

Write-Host "[protobuf] 清理旧文件..."
Remove-Item (Join-Path $out "pb") -Recurse -ErrorAction SilentlyContinue

Write-Host "[protobuf] 编译协议文件..."

$files = Get-ChildItem $src -Recurse *.proto
foreach ($file in $files) {
    $raw = Get-Content -Raw $file.FullName

    if ($raw -match 'go_package\s*=\s*"([^"]+)"') {
        $pkg = $matches[1]
    } else {
        Write-Host "  [skip] $($file.Name) no go_package"
        continue
    }

    Write-Host "  [编译] $($file.Name) -> $pkg"

    protoc --proto_path="$src" --go_out="module=${module}:${out}" $file.FullName

    if ($raw -match 'service\s+\w+') {
        Write-Host "  [gRPC] $($file.Name) stub"
        protoc --proto_path="$src" --go-grpc_out="module=${module}:${out}" $file.FullName
    }
}

Write-Host "[protobuf] done"
