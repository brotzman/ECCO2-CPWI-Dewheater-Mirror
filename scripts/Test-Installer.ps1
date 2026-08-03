[CmdletBinding()]
param(
    [ValidateSet('Setup','Msi')][string]$Mode,
    [string]$ArtifactRoot,
    [string]$ProductVersion = '1.0.1'
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath($ArtifactRoot)
$dist=Join-Path $root 'dist'
$logs=Join-Path $root 'test-logs'
New-Item -ItemType Directory -Force -Path $logs | Out-Null
$installDir=Join-Path $env:ProgramFiles 'ECCO2 CPWI Dew Mirror'
$exe=Join-Path $installDir 'ECCO2CPWIDewMirror.exe'
$icon=Join-Path $installDir 'ECCO2CPWIDewMirror.ico'
$desktopShortcut=Join-Path ([Environment]::GetFolderPath('CommonDesktopDirectory')) 'ECCO2 CPWI Dew Mirror.lnk'
$startMenuShortcut=Join-Path ([Environment]::GetFolderPath('CommonPrograms')) 'ECCO2 CPWI Dew Mirror\ECCO2 CPWI Dew Mirror.lnk'
$setup=Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Setup.exe"
$msi=Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-x64.msi"
$msiexec=Join-Path $env:SystemRoot 'System32\msiexec.exe'
function Invoke-Bounded([string]$File,[string]$Arguments,[int]$Timeout,[string]$Description) {
    Write-Host "START: $Description"
    Write-Host "COMMAND: $File $Arguments"
    $psi=[Diagnostics.ProcessStartInfo]::new(); $psi.FileName=$File; $psi.Arguments=$Arguments; $psi.UseShellExecute=$false; $psi.CreateNoWindow=$true
    $p=[Diagnostics.Process]::Start($psi)
    if (-not $p.WaitForExit($Timeout*1000)) { & taskkill.exe /PID $p.Id /T /F | Out-Host; throw "$Description timeout after $Timeout seconds." }
    if ($p.ExitCode -notin @(0,3010)) { throw "$Description failed with exit code $($p.ExitCode)." }
}
function Quote([string]$s) { '"' + $s.Replace('"','\"') + '"' }
function Get-SafePropertyValue([object]$InputObject,[string]$PropertyName) {
    if ($null -eq $InputObject) { return $null }
    $property=$InputObject.PSObject.Properties[$PropertyName]
    if ($null -eq $property) { return $null }
    return $property.Value
}
function Get-ProductRegistrations {
    $paths=@(
        'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )
    @(Get-ItemProperty -Path $paths -ErrorAction SilentlyContinue | Where-Object {
        $displayName=Get-SafePropertyValue $_ 'DisplayName'
        $displayVersion=Get-SafePropertyValue $_ 'DisplayVersion'
        $displayName -eq 'ECCO2 CPWI Dew Mirror' -and $displayVersion -eq $ProductVersion
    })
}
function Get-ProductRegistrationCount {
    $registrations=@(Get-ProductRegistrations)
    return [int]$registrations.Count
}
function Verify-Installed {
    if (-not (Test-Path $exe -PathType Leaf)) { throw "Installed executable missing: $exe" }
    $expected=(Get-FileHash (Join-Path $dist 'ECCO2CPWIDewMirror.exe') -Algorithm SHA256).Hash
    $actual=(Get-FileHash $exe -Algorithm SHA256).Hash
    if ($expected -ne $actual) { throw 'Installed executable hash differs from built executable.' }
    if (-not (Test-Path $icon -PathType Leaf)) { throw "Installed icon missing: $icon" }
    foreach($name in @('README.md','README_DE.md','CHANGELOG.md','LICENSE')) {
        $document=Join-Path $installDir $name
        if (-not (Test-Path $document -PathType Leaf)) { throw "Installed document missing: $document" }
    }
    if (-not (Test-Path $desktopShortcut -PathType Leaf)) { throw "Desktop shortcut missing: $desktopShortcut" }
    if (-not (Test-Path $startMenuShortcut -PathType Leaf)) { throw "Start-menu shortcut missing: $startMenuShortcut" }
    if ((Get-ProductRegistrationCount) -lt 1) { throw 'Uninstall registration missing.' }
    & (Join-Path $PSScriptRoot 'Test-GuiStartup.ps1') -Exe $exe -TimeoutSeconds 20 -DiagnosticDirectory $logs
}
function Verify-Uninstalled {
    if (Test-Path $installDir) { throw "Installation directory remains: $installDir" }
    if (Test-Path $desktopShortcut) { throw "Desktop shortcut remains: $desktopShortcut" }
    if (Test-Path $startMenuShortcut) { throw "Start-menu shortcut remains: $startMenuShortcut" }
    if ((Get-ProductRegistrationCount) -ne 0) { throw 'Uninstall registration remains.' }
}
function Wait-Uninstalled([int]$TimeoutSeconds = 90) {
    $deadline=[DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        $directoryExists=Test-Path $installDir
        $desktopExists=Test-Path $desktopShortcut
        $startMenuExists=Test-Path $startMenuShortcut
        $registrationCount=Get-ProductRegistrationCount
        if (-not $directoryExists -and -not $desktopExists -and -not $startMenuExists -and $registrationCount -eq 0) {
            return
        }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $deadline)
    Verify-Uninstalled
}
try {
    if ($Mode -eq 'Setup') {
        Invoke-Bounded $setup ("/quiet /norestart /log " + (Quote (Join-Path $logs 'setup-install.log'))) 180 'Setup installation'
        Verify-Installed
        Remove-Item $exe -Force
        Invoke-Bounded $setup ("/repair /quiet /norestart /log " + (Quote (Join-Path $logs 'setup-repair.log'))) 180 'Setup repair'
        Verify-Installed
        Invoke-Bounded $setup ("/uninstall /quiet /norestart /log " + (Quote (Join-Path $logs 'setup-uninstall.log'))) 180 'Setup uninstall'
    } else {
        Invoke-Bounded $msiexec ("/i " + (Quote $msi) + " /qn /norestart /L*V! " + (Quote (Join-Path $logs 'msi-install.log')) + " REBOOT=ReallySuppress") 180 'MSI installation'
        Verify-Installed
        Remove-Item $exe -Force
        Invoke-Bounded $msiexec ("/fa " + (Quote $msi) + " /qn /norestart /L*V! " + (Quote (Join-Path $logs 'msi-repair.log')) + " REBOOT=ReallySuppress") 180 'MSI repair'
        Verify-Installed
        Invoke-Bounded $msiexec ("/x " + (Quote $msi) + " /qn /norestart /L*V! " + (Quote (Join-Path $logs 'msi-uninstall.log')) + " REBOOT=ReallySuppress") 180 'MSI uninstall'
    }
    Wait-Uninstalled -TimeoutSeconds 90
    Write-Host "PASS: $Mode installer test completed."
} catch {
    Write-Host "FAIL: $Mode installer test: $($_.Exception.Message)"
    Write-Host "Install directory exists: $(Test-Path $installDir)"
    Write-Host "Desktop shortcut exists: $(Test-Path $desktopShortcut)"
    Write-Host "Start-menu shortcut exists: $(Test-Path $startMenuShortcut)"
    Write-Host "Matching uninstall registrations: $(Get-ProductRegistrationCount)"
    if (Test-Path $installDir) {
        Get-ChildItem -LiteralPath $installDir -Force -ErrorAction SilentlyContinue |
            ForEach-Object { Write-Host ("Installed item: {0} ({1} bytes)" -f $_.Name,$_.Length) }
    }
    throw
} finally {
    $needsCleanup=(Test-Path $installDir) -or ((Get-ProductRegistrationCount) -gt 0)
    if ($needsCleanup) {
        if ($Mode -eq 'Setup' -and (Test-Path $setup)) { try { Invoke-Bounded $setup '/uninstall /quiet /norestart' 90 'Cleanup setup uninstall'; Wait-Uninstalled -TimeoutSeconds 90 } catch {} }
        if ($Mode -eq 'Msi' -and (Test-Path $msi)) { try { Invoke-Bounded $msiexec ("/x " + (Quote $msi) + " /qn /norestart REBOOT=ReallySuppress") 90 'Cleanup MSI uninstall'; Wait-Uninstalled -TimeoutSeconds 90 } catch {} }
    }
}
