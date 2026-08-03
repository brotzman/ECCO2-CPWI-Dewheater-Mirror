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
function Verify-Installed {
    if (-not (Test-Path $exe -PathType Leaf)) { throw "Installed executable missing: $exe" }
    $expected=(Get-FileHash (Join-Path $dist 'ECCO2CPWIDewMirror.exe') -Algorithm SHA256).Hash
    $actual=(Get-FileHash $exe -Algorithm SHA256).Hash
    if ($expected -ne $actual) { throw 'Installed executable hash differs from built executable.' }
    $desktop=[Environment]::GetFolderPath('CommonDesktopDirectory')
    if (-not (Test-Path (Join-Path $desktop 'ECCO2 CPWI Dew Mirror.lnk'))) { throw 'Desktop shortcut missing.' }
    & (Join-Path $PSScriptRoot 'Test-GuiStartup.ps1') -Exe $exe -TimeoutSeconds 20
}
try {
    if ($Mode -eq 'Setup') {
        Invoke-Bounded $setup ("/quiet /norestart /log " + (Quote (Join-Path $logs 'setup-install.log'))) 180 'Setup installation'
        Verify-Installed
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
    Start-Sleep -Seconds 2
    if (Test-Path $installDir) { throw "Installation directory remains: $installDir" }
    Write-Host "PASS: $Mode installer test completed."
} finally {
    if (Test-Path $installDir) {
        if ($Mode -eq 'Setup' -and (Test-Path $setup)) { try { Invoke-Bounded $setup '/uninstall /quiet /norestart' 90 'Cleanup setup uninstall' } catch {} }
        if ($Mode -eq 'Msi' -and (Test-Path $msi)) { try { Invoke-Bounded $msiexec ("/x " + (Quote $msi) + " /qn /norestart REBOOT=ReallySuppress") 90 'Cleanup MSI uninstall' } catch {} }
    }
}
