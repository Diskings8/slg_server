$src = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol\src")
$out = Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol")
$module = "module=server.slg.com/api/protocol"
$commonM = "Mproto_common.proto=server.slg.com/api/protocol/pb/pb_common"

Write-Host "[protobuf] 编译所有协议文件..."

Get-ChildItem $src -Recurse *.proto | ForEach-Object {
    if ($_.Name -eq "game_server.proto") {
        # game_server.proto:
        #   --go_out         依赖 proto 内 go_package=pb_game  → 输出到 pb/pb_game/
        #   --go-grpc_out    M 参数覆盖为 pb_server           → 输出到 pb/pb_server/
        protoc --proto_path="$src" `
          --go_out="${commonM},module=server.slg.com/api/protocol:${out}" `
          --go-grpc_out="${commonM},Mservices/game_server.proto=server.slg.com/api/protocol/pb/pb_server,module=server.slg.com/api/protocol:${out}" `
          $_.FullName
    }
    else {
        # proto_common.proto: 纯 message，无 service
        protoc --proto_path="$src" `
          --go_out="${module}:${out}" `
          $_.FullName
    }
}

Write-Host "[protobuf] 完成"
