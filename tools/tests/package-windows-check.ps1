# One disposable GitHub Windows runner. Native acceptance runs as a separate user.
param([Parameter(Mandatory)][string]$TapRoot, [Parameter(Mandatory)][string]$Out)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repo = (Resolve-Path "$PSScriptRoot/../..").Path
$pins = Get-Content "$PSScriptRoot/package-windows-pins.json" -Raw | ConvertFrom-Json
if (Test-Path $Out) { throw "Evidence directory already exists: $Out" }
New-Item -ItemType Directory $Out | Out-Null
Start-Transcript -Path "$Out/setup.log" | Out-Null
$server = $null
$taskRegistered = $false
$timer = [Diagnostics.Stopwatch]::StartNew()
$phaseTimes = [ordered]@{}
try {
    if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_OS -ne 'Windows') {
        throw 'Run only on a disposable GitHub-hosted Windows runner.'
    }
    if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw 'This job requires Windows x64.' }
    $tapHead = (& git -C $TapRoot rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $tapHead -ne $pins.tap_commit) { throw 'Unexpected tap source revision.' }
    if (& git -C $TapRoot status --porcelain) { throw 'Tap inputs are dirty.' }
    Copy-Item "$PSScriptRoot/package-windows-pins.json" "$Out/pins.json"
    Copy-Item "$TapRoot/distribution/validator-pins.json" "$Out/schema-pins.json"
    $inputs = @(Get-ChildItem "$PSScriptRoot/package-windows-*" -File) +
              @(Get-Item "$repo/tools/install.ps1", "$PSScriptRoot/install-checksum-test.ps1",
                          "$repo/.github/workflows/package-windows-validation.yml") +
              @(Get-ChildItem "$TapRoot/distribution" -Recurse -File)
    $inputs | Get-FileHash -Algorithm SHA256 | Select-Object Path,Hash |
        ConvertTo-Json | Set-Content "$Out/inputs.json"
    $identity = [ordered]@{
        runnerImage = $env:ImageOS; runnerImageVersion = $env:ImageVersion
        runnerImageRelease = $env:ImageRelease; os = [Environment]::OSVersion.VersionString
        fixturePort = 18765
        patronusHead = (& git -C $repo rev-parse HEAD); tapHead = $tapHead
        githubSHA = $env:GITHUB_SHA; run = $env:GITHUB_RUN_ID; attempt = $env:GITHUB_RUN_ATTEMPT
        powerShell = $PSVersionTable.PSVersion.ToString(); python = (& python --version)
        go = (& go version); git = (& git --version)
        windowsArm = 'unrun'; lifecycle = 'later C/P'; publication = 'none'
    }
    $identity | ConvertTo-Json | Set-Content "$Out/runner.json"
    & python -m pip install --require-hashes --only-binary=:all: -r "$TapRoot/distribution/validation-requirements.txt"
    if ($LASTEXITCODE -ne 0) { throw 'Locked validator installation failed.' }
    & python -B "$PSScriptRoot/package-windows-prepare.py" --tap-root $TapRoot --out "$Out/fixtures"
    if ($LASTEXITCODE -ne 0) { throw 'Synthetic candidate preparation failed.' }
    $phaseTimes['prepared_at_seconds'] = $timer.Elapsed.TotalSeconds
    New-Item -ItemType Directory "$Out/downloads" | Out-Null
    foreach ($download in $pins.downloads.PSObject.Properties) {
        $path = "$Out/downloads/$($download.Name)"
        Invoke-WebRequest -Uri $download.Value.url -OutFile $path
        if ((Get-FileHash $path -Algorithm SHA256).Hash.ToLowerInvariant() -ne $download.Value.sha256) {
            throw "Dependency checksum mismatch: $($download.Name)"
        }
    }
    Expand-Archive "$Out/downloads/scoop.zip" "$Out/scoop-source"
    Expand-Archive "$Out/downloads/winget-dependencies.zip" "$Out/dependencies"
    $phaseTimes['downloaded_at_seconds'] = $timer.Elapsed.TotalSeconds
    # Only normal RemoteSigned checks; no execution-policy command-line override.
    Set-ExecutionPolicy RemoteSigned -Scope LocalMachine -Force
    & powershell.exe -NoProfile -Command 'Set-ExecutionPolicy RemoteSigned -Scope LocalMachine -Force'
    if ($LASTEXITCODE -ne 0) { throw 'Could not set the normal Windows PowerShell execution policy.' }
    Copy-Item "$PSScriptRoot/package-windows-user.ps1" "$Out/user.ps1"
    New-Item -ItemType Directory "$Out/source/tools/tests" -Force | Out-Null
    Copy-Item "$repo/tools/install.ps1" "$Out/source/tools/install.ps1"
    Copy-Item "$PSScriptRoot/install-checksum-test.ps1" "$Out/source/tools/tests/install-checksum-test.ps1"
    & powershell.exe -NoProfile -File "$Out/user.ps1" -Work $Out -Phase Register
    if ($LASTEXITCODE -ne 0) { throw 'Administrator WinGet registration failed.' }
    $winget = (Get-Content "$Out/admin-winget.txt" -Raw).Trim()
    & $winget --version
    if ($LASTEXITCODE -ne 0) { throw 'Pinned WinGet client unavailable.' }
    # WinGet admin settings are per SID. Machine policy reaches the normal user.
    $policy = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AppInstaller'
    New-Item $policy -Force | Out-Null
    New-ItemProperty $policy -Name EnableLocalManifestFiles -Value 1 -PropertyType DWord -Force | Out-Null
    Get-ItemProperty $policy -Name EnableLocalManifestFiles | Format-List | Out-String | Write-Output
    # Hosted images enable Developer Mode; ordinary-user acceptance requires it off.
    $developerPolicy = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock'
    New-Item $developerPolicy -Force | Out-Null
    New-ItemProperty $developerPolicy -Name AllowDevelopmentWithoutDevLicense `
        -Value 0 -PropertyType DWord -Force | Out-Null
    Get-ItemProperty $developerPolicy -Name AllowDevelopmentWithoutDevLicense |
        Select-Object AllowDevelopmentWithoutDevLicense | ConvertTo-Json |
        Set-Content "$Out/developer-mode.json"
    $phaseTimes['registered_at_seconds'] = $timer.Elapsed.TotalSeconds
    $server = Start-Process (Get-Command python).Source -ArgumentList @(
        '-m', 'http.server', '18765', '--bind', '127.0.0.1', '--directory', "$Out/fixtures/assets"
    ) -PassThru -RedirectStandardOutput "$Out/http.log" -RedirectStandardError "$Out/http-error.log"
    $ready = $false
    for ($i = 0; $i -lt 20; $i++) {
        try { Invoke-WebRequest 'http://127.0.0.1:18765/' | Out-Null; $ready = $true; break }
        catch { Start-Sleep -Milliseconds 500 }
    }
    if (-not $ready -or $server.HasExited) { throw 'Local fixture server unavailable.' }
    $name = 'patronusT3'
    if (Get-LocalUser $name -ErrorAction SilentlyContinue) { throw 'Test user already exists.' }
    $password = [guid]::NewGuid().ToString('N') + '!a9Z'
    $secure = ConvertTo-SecureString $password -AsPlainText -Force
    $user = New-LocalUser -Name $name -Password $secure -Description 'Disposable T3 normal-user acceptance'
    $usersGroup = Get-LocalGroup -SID 'S-1-5-32-545'
    Add-LocalGroupMember -Group $usersGroup -Member $user
    # Password-based tasks need this normal logon right; no administrator or
    # symlink privilege is granted. Preserve all existing assignees and denials.
    & secedit.exe /export /cfg "$Out/logon-before.inf" /areas USER_RIGHTS /log "$Out/logon-export.log"
    if ($LASTEXITCODE -ne 0) { throw 'Could not inspect existing batch-logon rights.' }
    $rights = Get-Content "$Out/logon-before.inf"
    $sid = $user.SID.Value
    $deny = @($rights | Where-Object { $_ -match '^SeDenyBatchLogonRight\s*=' }) -join ''
    foreach ($deniedSid in @($sid,'S-1-5-32-545','S-1-1-0','S-1-5-11')) {
        if ($deny -match [regex]::Escape($deniedSid)) { throw 'An explicit batch-logon denial applies; refusing to change it.' }
    }
    $assignment = @($rights | Where-Object { $_ -match '^SeBatchLogonRight\s*=' })
    $existing = if ($assignment.Count) { ($assignment[0] -split '=',2)[1].Trim() } else { '' }
    $assignees = @($existing -split ',' | Where-Object { $_ }) + "*$sid"
    @('[Unicode]','Unicode=yes','[Version]','signature="$CHICAGO$"','Revision=1',
        '[Privilege Rights]',('SeBatchLogonRight = ' + ($assignees -join ','))) |
        Set-Content "$Out/logon-grant.inf" -Encoding Unicode
    & secedit.exe /configure /db "$Out/logon-policy.sdb" /cfg "$Out/logon-grant.inf" `
        /areas USER_RIGHTS /log "$Out/logon-configure.log"
    if ($LASTEXITCODE -ne 0) { throw 'Could not assign the test account its batch-logon right.' }
    & icacls.exe $Out /grant '*S-1-5-32-545:(OI)(CI)M' /T /Q
    if ($LASTEXITCODE -ne 0) { throw 'Could not grant test-user access to staging evidence.' }
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument (
        '-NoProfile -File "{0}\user.ps1" -Work "{0}" -Phase All' -f $Out)
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Minutes 15) `
        -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    & wevtutil.exe sl Microsoft-Windows-TaskScheduler/Operational /e:true
    if ($LASTEXITCODE -ne 0) { throw 'Could not enable task startup diagnostics.' }
    $taskStarted = Get-Date
    Register-ScheduledTask -TaskName 'patronus-t3' -Action $action -Settings $settings `
        -User "$env:COMPUTERNAME\$name" -Password $password -RunLevel Limited | Out-Null
    $taskRegistered = $true
    Export-ScheduledTask -TaskName 'patronus-t3' | Set-Content "$Out/task-definition.txt"
    $password = $null
    Start-ScheduledTask 'patronus-t3'
    $deadline = (Get-Date).AddMinutes(16)
    $startupDeadline = (Get-Date).AddSeconds(30)
    do {
        Start-Sleep -Seconds 2
        $task = Get-ScheduledTask 'patronus-t3'
        $info = Get-ScheduledTaskInfo 'patronus-t3'
        if ($info.LastRunTime -lt $taskStarted.AddSeconds(-2) -and (Get-Date) -gt $startupDeadline) {
            throw "Test-user task never started (state $($task.State), result $($info.LastTaskResult))."
        }
    } while (($task.State -eq 'Running' -or $info.LastRunTime -lt $taskStarted.AddSeconds(-2)) -and (Get-Date) -lt $deadline)
    if ($task.State -eq 'Running') {
        Stop-ScheduledTask 'patronus-t3'
        for ($i = 0; $i -lt 20 -and (Get-ScheduledTask 'patronus-t3').State -eq 'Running'; $i++) {
            Start-Sleep -Milliseconds 500
        }
        throw 'Native user task exceeded its time bound; task was stopped.'
    }
    $info | Format-List | Out-String | Set-Content "$Out/task.txt"
    if ($task.State -eq 'Running' -or $info.LastTaskResult -ne 0 -or !(Test-Path "$Out/PASS.txt")) {
        throw 'Native user checks failed or timed out; inspect user.log and task.txt.'
    }
    Get-Content "$Out/user.log"
} finally {
    if ($taskRegistered) {
        if ((Get-ScheduledTask 'patronus-t3').State -in @('Running','Queued')) { Stop-ScheduledTask 'patronus-t3' }
        Get-ScheduledTaskInfo 'patronus-t3' | Format-List | Out-String | Set-Content "$Out/task.txt"
        $finalTask = Get-ScheduledTask 'patronus-t3'
        [ordered]@{
            task = $finalTask.TaskName; state = $finalTask.State.ToString()
            user = $finalTask.Principal.UserId; logonType = $finalTask.Principal.LogonType.ToString()
            runLevel = $finalTask.Principal.RunLevel.ToString()
            demandStart = $finalTask.Settings.AllowDemandStart
            disallowOnBattery = $finalTask.Settings.DisallowStartIfOnBatteries
        } | ConvertTo-Json | Set-Content "$Out/task-state.json"
        Get-WinEvent -FilterHashtable @{
            LogName = 'Microsoft-Windows-TaskScheduler/Operational'; StartTime = $taskStarted.AddSeconds(-5)
        } -ErrorAction SilentlyContinue | Where-Object { $_.Message -match 'patronus-t3' } |
            Select-Object TimeCreated,Id,LevelDisplayName,Message |
            ConvertTo-Json -Depth 4 | Set-Content "$Out/task-events.json"
    }
    $phaseTimes['finished_at_seconds'] = $timer.Elapsed.TotalSeconds
    $phaseTimes | ConvertTo-Json | Set-Content "$Out/phase-times.json"
    if ($null -ne $server -and !$server.HasExited) { Stop-Process -Id $server.Id }
    Stop-Transcript | Out-Null
}
