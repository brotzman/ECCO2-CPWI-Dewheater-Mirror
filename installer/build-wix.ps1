[CmdletBinding()]
param(
    [string]$Payload = '',
    [string]$Output = '',
    [string]$ProductVersion = '1.0.1',
    [string]$WixExe = 'wix.exe'
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$scriptDirectory = $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($scriptDirectory)) { $scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path }
if ([string]::IsNullOrWhiteSpace($Payload)) { $Payload = Join-Path $scriptDirectory 'payload' }
if ([string]::IsNullOrWhiteSpace($Output)) { $Output = Join-Path $scriptDirectory 'out' }
$payloadFull = [IO.Path]::GetFullPath($Payload)
$outputFull = [IO.Path]::GetFullPath($Output)
if (-not (Test-Path -LiteralPath $payloadFull -PathType Container)) { throw "Installer-Payload fehlt: $payloadFull" }
New-Item -ItemType Directory -Force -Path $outputFull | Out-Null
$wixCommand = Get-Command $WixExe -ErrorAction SilentlyContinue
if ($null -eq $wixCommand) {
    if (-not (Test-Path -LiteralPath $WixExe -PathType Leaf)) { throw "WiX wurde nicht gefunden: $WixExe" }
    $wixPath = [IO.Path]::GetFullPath($WixExe)
} else { $wixPath = $wixCommand.Source }
$packageSource = Join-Path $scriptDirectory 'wix\Package.wxs'
$msiPath = Join-Path $outputFull "ECCO2-CPWI-Dew-Mirror-$ProductVersion-x64.msi"
$intermediate = Join-Path $outputFull 'intermediate'
if (-not (Test-Path -LiteralPath $packageSource -PathType Leaf)) { throw "WiX-Quelldatei fehlt: $packageSource" }
& $wixPath --version
if ($LASTEXITCODE -ne 0) { throw 'WiX konnte nicht gestartet werden.' }
& $wixPath build $packageSource -arch x64 -d "Payload=$payloadFull" -d "ProductVersion=$ProductVersion" -intermediateFolder $intermediate -pdbtype none -o $msiPath
if ($LASTEXITCODE -ne 0) { throw 'MSI-Build fehlgeschlagen.' }
if (-not (Test-Path -LiteralPath $msiPath -PathType Leaf) -or (Get-Item $msiPath).Length -le 0) { throw "MSI fehlt: $msiPath" }
Remove-Item $intermediate -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $outputFull '.wix') -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "MSI erstellt: $msiPath"
