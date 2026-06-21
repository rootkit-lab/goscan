#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
$Version = (Get-Content (Join-Path $Root "assets\VERSION") -Raw).Trim()
$StageApp = Join-Path $Root "dist\package\msi\stage\app"
$OutDir = Join-Path $Root "dist"
$OutMsi = Join-Path $OutDir "goscan_${Version}_amd64.msi"
$HeatOut = Join-Path $Root "scripts\package\msi\Files.wxs"

Write-Host "=== GoScan MSI $Version ==="

if (-not (Get-Command wix -ErrorAction SilentlyContinue)) {
  throw "WiX CLI (wix) não encontrado — instala com: dotnet tool install --global wix"
}

# Build + stage (Git Bash no runner Windows)
$bash = "bash"
if (-not (Get-Command bash -ErrorAction SilentlyContinue)) {
  throw "bash em falta (Git for Windows)"
}

& $bash -lc "cd '$($Root -replace '\\','/')' && make release"
& $bash -lc "cd '$($Root -replace '\\','/')' && SKIP_RELEASE=1 ./scripts/package/stage.sh '$($StageApp -replace '\\','/')'"

Remove-Item -Force $HeatOut -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

wix extension add -g WixToolset.Util.wixext 2>$null

$stageForWix = $StageApp
wix heat dir $stageForWix `
  -cg AppFiles `
  -dr APPDIR `
  -srd -sfrag -gg `
  -var var.StageDir `
  -out $HeatOut

# ProductVersion: WiX exige x.x.x.x — normalizar
$parts = $Version.Split(".")
while ($parts.Count -lt 4) { $parts += "0" }
$ProductVersion = ($parts[0..3] -join ".")

wix build `
  (Join-Path $Root "scripts\package\msi\Product.wxs") `
  (Join-Path $Root "scripts\package\msi\Shortcuts.wxs") `
  $HeatOut `
  -ext WixToolset.Util.wixext `
  -d ProductVersion=$ProductVersion `
  -d StageDir=$stageForWix `
  -arch x64 `
  -o $OutMsi

Write-Host "✓ $OutMsi"
