param(
    [Parameter(Mandatory, Position=0)]
    [string]$TargetPath,

    [Parameter(Position=1)]
    [string]$SourcePath = (Resolve-Path (Join-Path $PSScriptRoot "..\api\protocol\src"))
)

if (-not (Test-Path $SourcePath)) {
    Write-Host "[proto-sync] ERROR: Source path not found: $SourcePath"
    exit 1
}

if (-not (Test-Path $TargetPath)) {
    New-Item -ItemType Directory -Path $TargetPath -Force | Out-Null
    Write-Host "[proto-sync] Created target directory: $TargetPath"
}

$files = Get-ChildItem $SourcePath -Filter *.proto
if ($files.Count -eq 0) {
    Write-Host "[proto-sync] No .proto files found in $SourcePath"
    exit 0
}

Write-Host "[proto-sync] $($files.Count) .proto files -> $TargetPath"

foreach ($file in $files) {
    $dest = Join-Path $TargetPath $file.Name
    Copy-Item $file.FullName $dest -Force
    Write-Host "  [copy] $($file.Name)"
}

Write-Host "[proto-sync] done"
