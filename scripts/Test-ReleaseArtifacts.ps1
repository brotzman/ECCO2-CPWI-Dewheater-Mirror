[CmdletBinding()]
param(
    [string]$DistRoot = (Join-Path $PSScriptRoot '..\dist'),
    [string]$ProductVersion = '1.0.1'
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$dist=[IO.Path]::GetFullPath($DistRoot)
$required=@(
    'BUILD_INFO.txt',
    'ECCO2CPWIDewMirror.exe',
    'ECCO2CPWIDewMirror.ico',
    "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Setup.exe",
    "ECCO2-CPWI-Dew-Mirror-$ProductVersion-x64.msi",
    "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Portable-Windows-x64.zip",
    "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Source.zip",
    'SHA256SUMS.txt'
)
foreach($name in $required) {
    $path=Join-Path $dist $name
    if(-not (Test-Path $path -PathType Leaf)) { throw "Release artifact missing: $path" }
}
$checksums=@{}
foreach($line in Get-Content (Join-Path $dist 'SHA256SUMS.txt')) {
    if($line -notmatch '^([0-9a-fA-F]{64})  (.+)$') { throw "Invalid checksum line: $line" }
    $checksums[$Matches[2]]=$Matches[1].ToLowerInvariant()
}
foreach($name in $required | Where-Object { $_ -ne 'SHA256SUMS.txt' }) {
    if(-not $checksums.ContainsKey($name)) { throw "Checksum missing for $name" }
    $actual=(Get-FileHash (Join-Path $dist $name) -Algorithm SHA256).Hash.ToLowerInvariant()
    if($checksums[$name] -ne $actual) { throw "Checksum mismatch for $name" }
}
Add-Type -AssemblyName System.IO.Compression.FileSystem
function Get-ZipEntries([string]$Path) {
    $zip=[IO.Compression.ZipFile]::OpenRead($Path)
    try { @($zip.Entries | ForEach-Object { $_.FullName.Replace('\','/') }) }
    finally { $zip.Dispose() }
}
$portable=Get-ZipEntries (Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Portable-Windows-x64.zip")
foreach($name in @('ECCO2CPWIDewMirror.exe','ECCO2CPWIDewMirror.ico','README.md','README_DE.md','CHANGELOG.md','LICENSE')) {
    if($name -notin $portable) { throw "Portable archive missing $name" }
}
$source=Get-ZipEntries (Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Source.zip")
foreach($name in @('.gitignore','.gitattributes','.editorconfig','LICENSE','go.mod','main_windows.go','assets/ECCO2CPWIDewMirror.ico','assets/ECCO2CPWIDewMirror.png')) {
    if($name -notin $source) { throw "Source archive missing $name" }
}
$forbiddenPrefixes=@('.git/','dist/','artifacts/','ci-input/','.config/','.wix/','installer/payload/','installer/out/','cmd/setup/payload/')
foreach($entry in $source) {
    foreach($prefix in $forbiddenPrefixes) {
        if($entry.StartsWith($prefix,[StringComparison]::OrdinalIgnoreCase)) { throw "Source archive contains generated/private path: $entry" }
    }
}
Write-Host 'PASS: release artifacts, checksums and archive contents validated.'
