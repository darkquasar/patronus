# Patronus installer for Windows (PowerShell).
#   iwr -useb https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.ps1 | iex
#   $env:PATRONUS_VERSION = "v2.7.0"; iwr -useb https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.ps1 | iex
#
# Downloads a release binary for this architecture (the latest unless
# PATRONUS_VERSION names one), verifies its sha256 against that release's
# checksums.txt, and installs it under %LOCALAPPDATA%\Programs\patronus, adding
# that dir to the user PATH.
$ErrorActionPreference = "Stop"

$repo = "darkquasar/patronus"
$version = if ($env:PATRONUS_VERSION) { $env:PATRONUS_VERSION } else { "latest" }
if ($version -eq "latest") {
  $base = "https://github.com/$repo/releases/latest/download"
} elseif ($version -match '^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$') {
  if ($version -notlike "v*") { $version = "v$version" }
  $base = "https://github.com/$repo/releases/download/$version"
} else {
  throw "invalid PATRONUS_VERSION: $version (want vX.Y.Z[-suffix] or latest)"
}

# --- detect arch ---
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "unsupported arch: $($env:PROCESSOR_ARCHITECTURE)" }
}
$bin = "patronus-windows-$arch.exe"

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([guid]::NewGuid()))
try {
  Write-Host "Downloading $bin ($version)..."
  Invoke-WebRequest -UseBasicParsing -Uri "$base/$bin" -OutFile "$tmp\patronus.exe"
  Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

  # --- verify sha256 ---
  $want = (Select-String -Path "$tmp\checksums.txt" -Pattern " $bin$").Line.Split()[0]
  if (-not $want) { throw "no checksum found for $bin" }
  $got = (Get-FileHash -Algorithm SHA256 "$tmp\patronus.exe").Hash.ToLower()
  if ($got -ne $want.ToLower()) { throw "checksum mismatch (got $got, want $want)" }

  # --- install ---
  $dest = Join-Path $env:LOCALAPPDATA "Programs\patronus"
  New-Item -ItemType Directory -Force -Path $dest | Out-Null
  Move-Item -Force "$tmp\patronus.exe" "$dest\patronus.exe"

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$dest", "User")
    Write-Host "Added $dest to your user PATH (restart your shell to use it)."
  }
  Write-Host "Installed patronus to $dest\patronus.exe"
}
finally {
  Remove-Item -Recurse -Force $tmp
}
