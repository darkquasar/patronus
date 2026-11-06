# Invoked by Windows PowerShell with normal policy, under a limited scheduled task.
param(
    [Parameter(Mandatory)][string]$Work,
    [ValidateSet('Register','All','Fresh')][string]$Phase = 'All',
    [ValidateSet('scoop','winget')][string]$Manager = 'scoop'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$pins = Get-Content "$Work/pins.json" -Raw | ConvertFrom-Json
$script:nativeRecords = @()
$recordName = if ($Phase -eq 'Fresh') { "fresh-$Manager" } else { $Phase }

function Invoke-Native {
    param([string]$Step, [string]$File, [string[]]$Arguments, [switch]$AllowFailure)
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $previous = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $lines = & $File @Arguments 2>&1 | Tee-Object -FilePath "$Work/$Step.log"
        $code = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previous }
    $text = ($lines | Out-String).Trim()
    $text | Write-Host
    $script:nativeRecords += [ordered]@{
        step = $Step; file = $File; arguments = $Arguments; exit = $code
        elapsed_seconds = $timer.Elapsed.TotalSeconds
    }
    $script:nativeRecords | ConvertTo-Json -Depth 5 | Set-Content "$Work/native-exits-$recordName.json"
    if ($code -ne 0 -and !$AllowFailure) { throw "Native step $Step failed (exit $code); see $Step.log." }
    return [pscustomobject]@{ Exit = $code; Text = $text }
}

function Assert-NormalUser {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    if ($principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator) -or
        $identity.Groups.Value -contains 'S-1-5-32-544') { throw 'Acceptance identity is an administrator.' }
    $privileges = & whoami.exe /priv
    if ($LASTEXITCODE -ne 0 -or ($privileges -join "`n") -match 'SeCreateSymbolicLinkPrivilege') {
        throw 'Normal-user check requires no symbolic-link privilege.'
    }
    $developer = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock' `
        -Name AllowDevelopmentWithoutDevLicense -ErrorAction SilentlyContinue
    if ($developer -and $developer.AllowDevelopmentWithoutDevLicense -ne 0) {
        throw 'Developer Mode is enabled; this host cannot prove the required ordinary-user alias behavior.'
    }
    $policies = @(Get-ExecutionPolicy -List)
    if ((Get-ExecutionPolicy) -ne 'RemoteSigned' -or
        @($policies | Where-Object { $_.ExecutionPolicy -in @('Bypass','Unrestricted') }).Count) {
        throw 'Acceptance requires RemoteSigned policy with no permissive override scope.'
    }
    Write-Output "Identity: $($identity.Name) $($identity.User.Value); non-admin, no symlink privilege, Developer Mode off."
    $privileges | Write-Output
}

function Register-WinGet {
    # Dependencies are from the same hash-locked official release archive.
    $dependencies = @(Get-ChildItem "$Work/dependencies" -Recurse -Filter '*.appx' |
        Where-Object { $_.FullName -match '[\\/]x64[\\/]' })
    if ($dependencies.Count -ne 3) { throw 'Expected three pinned x64 WinGet runtime dependencies.' }
    Add-AppxPackage -Path "$Work/downloads/winget.msixbundle" -DependencyPath $dependencies.FullName
    $packages = @(Get-AppxPackage -Name Microsoft.DesktopAppInstaller |
        Where-Object { $_.Architecture -eq 'X64' })
    if ($packages.Count -ne 1) { throw 'Expected one registered x64 DesktopAppInstaller package.' }
    $packages | Select-Object Name,Version,Architecture,PackageFullName |
        ConvertTo-Json | Set-Content "$Work/appx-$recordName.json"
    $exe = "$env:LOCALAPPDATA\Microsoft\WindowsApps\winget.exe"
    if (!(Test-Path $exe)) { throw 'Registered user WinGet App Execution Alias is unavailable.' }
    $version = Invoke-Native -Step "winget-version-$recordName" -File $exe -Arguments @('--version')
    if ($version.Text -ne $pins.winget_version) {
        throw "Unexpected WinGet client: $($version.Text)"
    }
    return $exe
}

if ($Phase -eq 'Register') {
    Register-WinGet | Set-Content "$Work/admin-winget.txt"
    exit 0
}

if ($Phase -eq 'Fresh') {
    Assert-NormalUser
    # Prove persisted registry PATH resolution in a new process, not a new logon.
    $env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' +
                [Environment]::GetEnvironmentVariable('Path','User')
    foreach ($app in @('patronus','lumen')) {
        $snapshot = Get-Content "$Work/fixtures/$app-snapshot.json" -Raw | ConvertFrom-Json
        $command = Get-Command "$app.exe" -CommandType Application
        $registration = Get-Content "$Work/fixtures/$app-registration.json" -Raw | ConvertFrom-Json
        $mechanism = 'scoop-shim'
        if ($Manager -eq 'scoop') {
            if ($command.Source -ne "$env:USERPROFILE\scoop\shims\$app.exe") {
                throw "Unexpected Scoop executable resolution: $($command.Source)"
            }
        } else {
            $links = "$env:LOCALAPPDATA\Microsoft\WinGet\Links\$app.exe"
            $packageRoot = "$env:LOCALAPPDATA\Microsoft\WinGet\Packages\"
            $identifier = $registration.configuration.identifiers.winget
            $validPackage = $false
            if ($command.Source.StartsWith($packageRoot, [StringComparison]::OrdinalIgnoreCase)) {
                $parts = $command.Source.Substring($packageRoot.Length).Split('\')
                $validPackage = $parts.Count -eq 2 -and $parts[0].StartsWith(
                    "$($identifier)_", [StringComparison]::OrdinalIgnoreCase) -and $parts[1] -eq "$app.exe"
            }
            if ($command.Source -eq $links) { $mechanism = 'winget-link' }
            elseif ($validPackage) { $mechanism = 'winget-package-path' }
            else { throw "Unexpected WinGet executable resolution: $($command.Source)" }
            $asset = $registration.configuration.assets.'windows-amd64'
            if ((Get-FileHash $command.Source -Algorithm SHA256).Hash.ToLowerInvariant() -ne
                $snapshot.payload.identity.assets.$asset.sha256) { throw 'Resolved WinGet asset hash differs.' }
        }
        $actual = Invoke-Native -Step "$Manager-version-$app" -File $command.Source -Arguments @('--version')
        if ($actual.Text -ne "$app $($snapshot.payload.identity.version)") {
            throw "Wrong exposed version: $($actual.Text)"
        }
        Write-Host "$Manager persisted-PATH check: $mechanism; $($command.Source) => $($actual.Text)"
    }
    exit 0
}

Start-Transcript "$Work/user.log" | Out-Null
try {
    Assert-NormalUser
    Get-ExecutionPolicy -List | Format-Table | Out-String | Write-Output
    $null = Invoke-Native -Step 'installer-checksum' -File 'powershell.exe' -Arguments @(
        '-NoProfile','-File',"$Work/source/tools/tests/install-checksum-test.ps1")
    # Install pinned Scoop from source using the unchanged official bootstrap's
    # setup functions. Do not run Install-Scoop: it fetches mutable master refs.
    $env:SCOOP = "$env:USERPROFILE\scoop"
    & {
        Set-StrictMode -Off
        $env:SCOOP_NOINSTALL = 'true'
        . "$Work/downloads/scoop-install.ps1" -ScoopDir $env:SCOOP
        $source = "$Work/scoop-source/Scoop-$($pins.scoop_commit)"
        New-Item -ItemType Directory "$env:SCOOP/apps/scoop/current" -Force | Out-Null
        Copy-Item "$source/*" "$env:SCOOP/apps/scoop/current" -Recurse
        Import-ScoopShim
        Add-ShimsDirToPath
        Add-DefaultConfig
    }
    $env:SCOOP_NOINSTALL = $null
    $scoop = "$env:SCOOP/apps/scoop/current/bin/scoop.ps1"
    # Use the .ps1 entry with normal policy. Upstream .cmd/.sh launchers contain
    # their own policy override; this harness never invokes them.
    $null = Invoke-Native -Step 'scoop-config' -File 'powershell.exe' -Arguments @(
        '-NoProfile','-File',$scoop,'config','aria2-enabled','false')
    $null = Invoke-Native -Step 'user-git-bytes' -File 'git.exe' -Arguments @('config','--global','core.autocrlf','false')
    New-Item -ItemType Directory "$env:SCOOP/buckets" -Force | Out-Null
    $bucket = "$env:USERPROFILE\t3-synthetic-bucket"
    New-Item -ItemType Directory $bucket | Out-Null
    Copy-Item "$Work/fixtures/local-transport/scoop/*" $bucket -Recurse
    foreach ($entry in @(
        @{ step='bucket-init'; arguments=@('-C',$bucket,'init','--initial-branch=main') },
        @{ step='bucket-add'; arguments=@('-C',$bucket,'add','bucket') },
        @{ step='bucket-commit'; arguments=@('-C',$bucket,'-c','user.name=T3 Fixture',
            '-c','user.email=fixture@example.invalid','commit','-m','Synthetic two-app staging bucket') }
    )) { $null = Invoke-Native -Step $entry.step -File 'git.exe' -Arguments $entry.arguments }
    $bucketUri = ([uri]$bucket).AbsoluteUri
    $null = Invoke-Native -Step 'scoop-bucket-register' -File 'powershell.exe' -Arguments @(
        '-NoProfile','-File',$scoop,'bucket','add','darkquasar',$bucketUri)
    $apps = @('patronus','lumen')
    foreach ($app in $apps) {
        $localManifest = "$Work/fixtures/local-transport/scoop/bucket/$app.json"
        $registeredManifest = "$env:SCOOP/buckets/darkquasar/bucket/$app.json"
        if (!(Test-Path $registeredManifest) -or
            (Get-FileHash $registeredManifest).Hash -ne (Get-FileHash $localManifest).Hash) {
            throw "Registered bucket manifest differs from staged bytes: $app"
        }
        $info = Invoke-Native -Step "scoop-info-$app" -File 'powershell.exe' -Arguments @(
            '-NoProfile','-File',$scoop,'info',"darkquasar/$app")
        if ($info.Text -notmatch $app -or $info.Text -notmatch 'darkquasar') { throw "Scoop discovery failed: $app" }
        $denial = Invoke-Native -Step "scoop-x86-$app" -File 'powershell.exe' -AllowFailure -Arguments @(
            '-NoProfile','-File',$scoop,'install',"darkquasar/$app",'--arch','32bit','--no-update-scoop')
        # This pinned Scoop prints an error then returns exit 0 for unsupported
        # architecture. Require its exact denial and absence of installed files.
        if ($denial.Text -notmatch "doesn't support current architecture" -or
            (Test-Path "$env:SCOOP/apps/$app") -or (Test-Path "$env:SCOOP/shims/$app.exe")) {
            throw "x86 installation was not denied: $app"
        }
        $null = Invoke-Native -Step "scoop-install-$app" -File 'powershell.exe' -Arguments @(
            '-NoProfile','-File',$scoop,'install',"darkquasar/$app",'--arch','64bit','--no-update-scoop')
    }
    $null = Invoke-Native -Step 'fresh-scoop' -File 'powershell.exe' -Arguments @(
        '-NoProfile','-File',"$Work/user.ps1",'-Work',$Work,'-Phase','Fresh','-Manager','scoop')
    foreach ($app in $apps) {
        $null = Invoke-Native -Step "scoop-uninstall-$app" -File 'powershell.exe' -Arguments @(
            '-NoProfile','-File',$scoop,'uninstall',$app)
        if (Test-Path "$env:SCOOP/shims/$app.exe") { throw "Scoop test uninstall failed: $app" }
    }
    $winget = Register-WinGet
    Get-AppxPackage | Where-Object { $_.Name -in @('Microsoft.DesktopAppInstaller',
        'Microsoft.VCLibs.140.00', 'Microsoft.VCLibs.140.00.UWPDesktop', 'Microsoft.WindowsAppRuntime.1.8') } |
        Select-Object Name,Version,Architecture,PackageFullName |
        ConvertTo-Json | Set-Content "$Work/installed-runtime.json"
    $null = Invoke-Native -Step 'winget-info' -File $winget -Arguments @('--info')
    $policy = Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AppInstaller' -Name EnableLocalManifestFiles
    if ($policy.EnableLocalManifestFiles -ne 1) { throw 'Machine LocalManifestFiles policy is not effective.' }
    foreach ($app in $apps) {
        $snapshot = Get-Content "$Work/fixtures/$app-snapshot.json" -Raw | ConvertFrom-Json
        $registration = Get-Content "$Work/fixtures/$app-registration.json" -Raw | ConvertFrom-Json
        $relative = $registration.configuration.destinations.winget.path + '/' + $snapshot.payload.identity.version
        $identifier = $registration.configuration.identifiers.winget
        foreach ($kind in @('canonical','local-transport')) {
            $directory = "$Work/fixtures/$kind/winget/$relative"
            $null = Invoke-Native -Step "winget-validate-$app-$kind" -File $winget -Arguments @('validate','--manifest',$directory)
        }
        $manifest = "$Work/fixtures/local-transport/winget/$relative"
        $beforePath = [Environment]::GetEnvironmentVariable('Path','User')
        $denial = Invoke-Native -Step "winget-x86-$app" -File $winget -AllowFailure -Arguments @(
            'install','--manifest',$manifest,'--scope','user','--architecture','x86',
            '--accept-package-agreements','--accept-source-agreements','--disable-interactivity')
        if ($denial.Exit -eq 0 -or $denial.Text -notmatch 'No applicable installer' -or
            (Test-Path "$env:LOCALAPPDATA\Microsoft\WinGet\Packages\$($identifier)_*") -or
            (Test-Path "$env:LOCALAPPDATA\Microsoft\WinGet\Links\$app.exe") -or
            [Environment]::GetEnvironmentVariable('Path','User') -ne $beforePath) {
            throw "WinGet x86 denial failed: $app"
        }
        $installArguments = @('install','--manifest',$manifest,'--architecture','x64',
            '--accept-package-agreements','--accept-source-agreements','--disable-interactivity')
        # Check the documented default for Patronus and explicit per-user selection
        # for the second app. Fresh-phase paths and hashes must prove user scope.
        if ($app -eq 'lumen') { $installArguments += @('--scope','user') }
        $null = Invoke-Native -Step "winget-install-$app" -File $winget -Arguments $installArguments
    }
    $null = Invoke-Native -Step 'fresh-winget' -File 'powershell.exe' -Arguments @(
        '-NoProfile','-File',"$Work/user.ps1",'-Work',$Work,'-Phase','Fresh','-Manager','winget')
    'PASS: two synthetic apps; Windows x64 normal-user checks only. ARM, live release and C/P lifecycle unrun.' |
        Set-Content "$Work/PASS.txt"
} catch {
    $_ | Format-List * -Force | Out-String | Set-Content "$Work/failure.txt"
    throw
} finally {
    $logs = @(
        "$env:LOCALAPPDATA\Packages\Microsoft.DesktopAppInstaller_8wekyb3d8bbwe\LocalState\DiagOutputDir",
        "$env:LOCALAPPDATA\Microsoft\WinGet\DiagOutputDir", "$env:TEMP\WinGet"
    )
    for ($i = 0; $i -lt $logs.Count; $i++) {
        if (Test-Path $logs[$i]) {
            New-Item -ItemType Directory "$Work/winget-logs/$i" -Force | Out-Null
            Copy-Item "$($logs[$i])/*" "$Work/winget-logs/$i" -Recurse -ErrorAction Continue
        }
    }
    Stop-Transcript | Out-Null
}
