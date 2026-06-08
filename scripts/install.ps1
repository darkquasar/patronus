# Patronus installer for Windows (PowerShell).
#   iwr -useb https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.ps1 | iex
#
# Downloads the latest release binary for this architecture, verifies its sha256
# against the published checksums.txt, and installs it under
# %LOCALAPPDATA%\Programs\patronus, adding that dir to the user PATH.
$ErrorActionPreference = "Stop"

$repo = "darkquasar/patronus"
$base = "https://github.com/$repo/releases/latest/download"

# --- detect arch ---
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "unsupported arch: $($env:PROCESSOR_ARCHITECTURE)" }
}
$bin = "patronus-windows-$arch.exe"

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([guid]::NewGuid()))
try {
  Write-Host "Downloading $bin..."
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
