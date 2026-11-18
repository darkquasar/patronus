# Real installer; only Invoke-WebRequest is replaced. Rejection precedes PATH writes.
$ErrorActionPreference = 'Stop'
$installer = Join-Path $PSScriptRoot '../install.ps1'
$work = Join-Path ([IO.Path]::GetTempPath()) ('patronus-checksum-' + [guid]::NewGuid())
$keys = @('TEMP','LOCALAPPDATA','PROCESSOR_ARCHITECTURE','PATRONUS_VERSION')
$saved = @{}
foreach ($key in $keys) { $saved[$key] = [Environment]::GetEnvironmentVariable($key,'Process') }
$onWindows = [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT
$priorUserPath = if ($onWindows) { [Environment]::GetEnvironmentVariable('Path','User') } else { $null }
try {
    New-Item -ItemType Directory -Path $work | Out-Null
    $env:TEMP = Join-Path $work 'temp'
    $env:LOCALAPPDATA = Join-Path $work 'local'
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    New-Item -ItemType Directory -Path $env:TEMP | Out-Null
    $dest = Join-Path $env:LOCALAPPDATA 'Programs/patronus'
    New-Item -ItemType Directory -Path $dest -Force | Out-Null
    $sentinel = Join-Path $dest 'patronus.exe'
    [IO.File]::WriteAllText($sentinel,'existing user binary')
    $before = (Get-FileHash $sentinel).Hash
    function Invoke-WebRequest {
        param([switch]$UseBasicParsing, [string]$Uri, [string]$OutFile)
        $requests.Add($Uri)
        $OutFile = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutFile)
        if ($Uri.EndsWith('/checksums.txt')) {
            $name = if ($checksumCase -eq 'missing') { 'other.exe' } else { 'patronus-windows-amd64.exe' }
            [IO.File]::WriteAllText($OutFile, (('0' * 64) + '  ' + $name + "`n"))
        } else {
            [IO.File]::WriteAllText($OutFile,'invented binary bytes')
        }
    }
    foreach ($kind in @('mismatch','missing')) {
        foreach ($version in @('latest','v2.7.1','v2.8.0-docs.1')) {
            $checksumCase = $kind
            $requests = [Collections.Generic.List[string]]::new()
            $env:PATRONUS_VERSION = $version
            $denied = $false
            try { & $installer *> $null } catch {
                $denied = $true
                if ($kind -eq 'mismatch' -and $_.Exception.Message -notlike '*checksum mismatch*') { throw }
                # Characterize the existing missing-checksum diagnostic; no installer change here.
                if ($kind -eq 'missing' -and $_.FullyQualifiedErrorId -ne 'InvokeMethodOnNull') { throw }
            }
            if (-not $denied) { throw "Checksum case $kind was not rejected" }
            if ($requests.Count -ne 2) { throw "Expected both binary and checksum requests, got $($requests.Count)" }
            $base = if ($version -eq 'latest') { 'https://github.com/darkquasar/patronus/releases/latest/download' } else { "https://github.com/darkquasar/patronus/releases/download/$version" }
            if ($requests[0] -ne "$base/patronus-windows-amd64.exe" -or $requests[1] -ne "$base/checksums.txt") { throw 'Installer URL mismatch' }
            if ((Get-FileHash $sentinel).Hash -ne $before) { throw 'Existing executable changed' }
            if ($onWindows -and [Environment]::GetEnvironmentVariable('Path','User') -ne $priorUserPath) { throw 'User PATH changed' }
        }
    }
    Write-Host 'install.ps1 checksum rejection reasons, URL selection and user sentinel: ok'
    Write-Host 'Missing checksum is characterized as InvokeMethodOnNull in the existing installer.'
    if ($onWindows) { Write-Host 'Windows User PATH preservation: ok' }
    else { Write-Host 'Windows User PATH preservation: not verified on this platform (Windows host pending)' }
} finally {
    foreach ($key in $keys) { [Environment]::SetEnvironmentVariable($key,$saved[$key],'Process') }
    if (Test-Path $work) { Remove-Item -LiteralPath $work -Recurse -Force }
}
