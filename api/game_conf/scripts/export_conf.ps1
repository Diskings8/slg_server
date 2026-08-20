# One-shot config export: xlsx sources -> gameconfig.json + gameconfig.proto + pb.go
#
# Usage: pwsh -File api/game_conf/scripts/export_conf.ps1
# NOTE: keep this file ASCII-only (Windows PowerShell 5.1 parses BOM-less files as ANSI).
# Outputs:
#   api/game_conf/json/gameconfig.json          (tabtoy single JSON)
#   api/protocol/src/gameconfig.proto           (tabtoy generated + go_package injected)
#   api/protocol/pb/pb_gameconfig/*.pb.go       (protoc compiled)
$ErrorActionPreference = 'Stop'

$scripts = $PSScriptRoot
$gameConf = Join-Path $scripts '..'
$executor = Join-Path $gameConf 'executor'
$excel = Join-Path $gameConf '..\data_excel'
$jsonOut = Join-Path $gameConf 'json\gameconfig.json'
$protoOut = Join-Path $gameConf '..\protocol\src\gameconfig.proto'
$buildS = Join-Path $gameConf '..\..\scripts\build_s.ps1'

Write-Host '[1/3] Generating config files using tabtoy...'
# tabtoy resolves relative filenames inside Index.xlsx against cwd, so run from excel dir.
Push-Location $excel
try {
    # NOTE: PS5.1 does not expand $var in -flag=$var native args; use double quotes.
    & (Join-Path $executor 'tabtoy.exe') `
        -mode=v3 `
        -package=pb_gameconfig `
        -index="Index.xlsx" `
        -json_out="$jsonOut" `
        -proto_out="$protoOut"
    if ($LASTEXITCODE -ne 0) { throw "tabtoy failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host '[2/3] Injecting go_package into gameconfig.proto...'
& python "$executor\gameconfig.py"
if ($LASTEXITCODE -ne 0) { throw "go_package injection failed" }

Write-Host '[3/3] Compiling protobuf...'
& powershell -File $buildS
if ($LASTEXITCODE -ne 0) { throw "protobuf compile failed" }

Write-Host 'Export completed successfully.'
