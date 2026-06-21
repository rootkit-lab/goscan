#Requires -Version 5.1
$ErrorActionPreference = "Stop"

function Invoke-Checked {
  param([scriptblock]$Block)
  & @Block
  if ($LASTEXITCODE -ne 0) { throw "Command failed with exit code $LASTEXITCODE" }
}

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
$Version = (Get-Content (Join-Path $Root "assets\VERSION") -Raw).Trim()
$StageApp = Join-Path $Root "dist\package\msi\stage\app"
$OutDir = Join-Path $Root "dist"
$OutMsi = Join-Path $OutDir "goscan_${Version}_amd64.msi"
$HeatOut = Join-Path $Root "scripts\package\msi\Files.wxs"
$ProductWxs = Join-Path $Root "scripts\package\msi\Product.wxs"
$WixObj = Join-Path $Root "dist\package\msi\wixobj"

Write-Host "=== GoScan MSI $Version ==="

$bash = "bash"
if (-not (Get-Command bash -ErrorAction SilentlyContinue)) {
  throw "bash em falta (Git for Windows)"
}

$rootUnix = ($Root -replace '\\', '/')
Invoke-Checked { & $bash -lc "cd '$rootUnix' && chmod +x scripts/package/build-release.sh && ./scripts/package/build-release.sh" }
Invoke-Checked { & $bash -lc "cd '$rootUnix' && SKIP_RELEASE=1 ./scripts/package/stage.sh '$($StageApp -replace '\\', '/')'" }

if (-not (Test-Path (Join-Path $StageApp "bin\goscan-ui.exe"))) {
  throw "Staging incompleto: goscan-ui.exe em falta"
}

$heat = Get-Command heat.exe -ErrorAction SilentlyContinue
$candle = Get-Command candle.exe -ErrorAction SilentlyContinue
$light = Get-Command light.exe -ErrorAction SilentlyContinue
if (-not $heat -or -not $candle -or -not $light) {
  throw "WiX Toolset 3.x em falta (heat/candle/light) — instala com: choco install wixtoolset"
}

Remove-Item -Force $HeatOut -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $WixObj -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $OutDir, $WixObj | Out-Null

$parts = $Version.Split(".")
while ($parts.Count -lt 4) { $parts += "0" }
$ProductVersion = ($parts[0..3] -join ".")

& heat.exe dir "$StageApp" `
  -cg AppFiles `
  -dr APPDIR `
  -srd -sfrag -gg `
  -platform x64 `
  -var var.StageDir `
  -out $HeatOut
if ($LASTEXITCODE -ne 0) { throw "heat falhou" }

& candle.exe -nologo -arch x64 `
  "-dProductVersion=$ProductVersion" `
  "-dStageDir=$StageApp" `
  -out "$WixObj\" `
  $ProductWxs $HeatOut
if ($LASTEXITCODE -ne 0) { throw "candle falhou" }

& light.exe -nologo `
  -out $OutMsi `
  (Join-Path $WixObj "Product.wixobj") `
  (Join-Path $WixObj "Files.wixobj")
if ($LASTEXITCODE -ne 0) { throw "light falhou" }

if (-not (Test-Path $OutMsi)) {
  throw "MSI não gerado: $OutMsi"
}

Write-Host "✓ $OutMsi"
