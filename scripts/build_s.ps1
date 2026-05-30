$src = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol\src")
$out = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol")

Write-Host "[protobuf] 编译所有协议文件..."

Get-ChildItem $src -Recurse *.proto | ForEach-Object {
    protoc --proto_path="$src" --go_out="$out" --go-grpc_out="$out" $_.FullName
}

Write-Host "[protobuf] 完成"
