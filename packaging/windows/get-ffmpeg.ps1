<#
  get-ffmpeg.ps1 - download ffmpeg, ffplay and their license into build/windows/redist,
  where `task backend:build:windows` copies the two programs from on its way to build/bin.

  Not straight into build/bin: that directory is a build output and is rebuilt,
  while the redistributables are fetched once per machine and kept (docs/packaging.md).
  The app looks for the pair beside its own executable and then on PATH
  (backend/internal/ffmpeg, FindExe), which is what the copy into build/bin serves.

  A full GPL build, because ddagrab and ffplay are both needed and the smaller builds drop one or the other.
  Its license text has to travel with the zip (THIRD-PARTY-NOTICES.md),
  and packaging/windows/package.ps1 picks it up as ffmpeg-LICENSE.txt.
  The release workflow runs this script too, so a local zip and a released one carry the same build.

  Run once per machine:
    .\get-ffmpeg.ps1
#>
param(
  [string]$Dest = (Join-Path (Split-Path (Split-Path $PSScriptRoot -Parent) -Parent) "build\windows\redist")
)

$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force $Dest | Out-Null
$zip = Join-Path $Dest "ffmpeg.zip"
$tmp = Join-Path $Dest "_x"
$url = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

Write-Host "downloading $url ..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
Expand-Archive -Path $zip -DestinationPath $tmp -Force
foreach ($exe in "ffmpeg.exe", "ffplay.exe") {
  $found = Get-ChildItem -Path $tmp -Recurse -Filter $exe | Select-Object -First 1
  if (-not $found) { throw "$exe is not in the downloaded build" }
  Copy-Item $found.FullName $Dest -Force
}
$license = Get-ChildItem -Path $tmp -Recurse -Filter LICENSE.txt | Select-Object -First 1
if (-not $license) { throw "the downloaded ffmpeg build carries no LICENSE.txt" }
Copy-Item $license.FullName (Join-Path $Dest "ffmpeg-LICENSE.txt") -Force
Remove-Item $zip -Force
Remove-Item $tmp -Recurse -Force
Write-Host "ffmpeg, ffplay and ffmpeg-LICENSE.txt in $Dest" -ForegroundColor Green
