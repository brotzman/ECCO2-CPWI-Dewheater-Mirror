[CmdletBinding()]
param(
    [string]$ProductVersion = '1.0.1',
    [string]$WixVersion = '5.0.2',
    [string]$WixExe = 'wix.exe',
    [switch]$SkipMsi
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$dist = Join-Path $root 'dist'
$payload = Join-Path $root 'installer\payload'
$setupPayload = Join-Path $root 'cmd\setup\payload'
$wixOut = Join-Path $root 'installer\out'
Remove-Item $dist,$payload,$setupPayload,$wixOut -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist,$payload,$setupPayload,$wixOut | Out-Null
Push-Location $root
try {
    go version
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go-Tests fehlgeschlagen.' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet fehlgeschlagen.' }
    $env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
    go build -trimpath -buildvcs=false -ldflags '-s -w -H=windowsgui' -o (Join-Path $payload 'ECCO2CPWIDewMirror.exe') .
    if ($LASTEXITCODE -ne 0) { throw 'Windows-x64-Build fehlgeschlagen.' }
} finally { Pop-Location }
Copy-Item (Join-Path $root 'README.md'),(Join-Path $root 'README_DE.md'),(Join-Path $root 'CHANGELOG.md'),(Join-Path $root 'LICENSE'),(Join-Path $root 'assets\ECCO2CPWIDewMirror.ico') -Destination $payload
Copy-Item (Join-Path $payload '*') -Destination $setupPayload -Recurse -Force
Push-Location $root
try {
    go build -tags installerpayload -trimpath -buildvcs=false -ldflags '-s -w -H=windowsgui' -o (Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Setup.exe") ./cmd/setup
    if ($LASTEXITCODE -ne 0) { throw 'Native Setup-Build fehlgeschlagen.' }
} finally { Pop-Location }
Copy-Item (Join-Path $payload 'ECCO2CPWIDewMirror.exe'),(Join-Path $payload 'ECCO2CPWIDewMirror.ico') -Destination $dist
if (-not $SkipMsi) {
    & (Join-Path $root 'installer\build-wix.ps1') -Payload $payload -Output $wixOut -ProductVersion $ProductVersion -WixExe $WixExe
    Copy-Item (Join-Path $wixOut '*') -Destination $dist -Force
}
$portableDir = Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Portable-Windows-x64"
New-Item -ItemType Directory -Force -Path $portableDir | Out-Null
Copy-Item (Join-Path $payload '*') -Destination $portableDir -Recurse -Force
Compress-Archive -Path (Join-Path $portableDir '*') -DestinationPath "$portableDir.zip" -CompressionLevel Optimal
Remove-Item $portableDir -Recurse -Force
$sourceStage = Join-Path $env:TEMP ("ecco2-source-" + [guid]::NewGuid().ToString('N'))
$sourceZip = Join-Path $dist "ECCO2-CPWI-Dew-Mirror-$ProductVersion-Source.zip"
$sourceExcludes = @('.git','dist','artifacts','ci-input','.config','.wix')
New-Item -ItemType Directory -Force -Path $sourceStage | Out-Null
try {
    Get-ChildItem -LiteralPath $root -Force |
        Where-Object { $_.Name -notin $sourceExcludes } |
        Copy-Item -Destination $sourceStage -Recurse -Force
    Remove-Item (Join-Path $sourceStage 'installer\payload'),(Join-Path $sourceStage 'installer\out'),(Join-Path $sourceStage 'cmd\setup\payload') -Recurse -Force -ErrorAction SilentlyContinue
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [IO.Compression.ZipFile]::CreateFromDirectory($sourceStage,$sourceZip,[IO.Compression.CompressionLevel]::Optimal,$false)
} finally { Remove-Item $sourceStage -Recurse -Force -ErrorAction SilentlyContinue }
$commit=if ($env:GITHUB_SHA) {$env:GITHUB_SHA} else {'local'}
$buildInfo = @("Product: ECCO2 CPWI Dew Mirror","Version: $ProductVersion","Go: $(go version)","Commit: $commit","BuiltUTC: $([DateTime]::UtcNow.ToString('o'))") -join "`r`n"
[IO.File]::WriteAllText((Join-Path $dist 'BUILD_INFO.txt'),$buildInfo,[Text.UTF8Encoding]::new($false))
$checksumLines = Get-ChildItem -LiteralPath $dist -File | Where-Object Name -ne 'SHA256SUMS.txt' | Sort-Object Name | ForEach-Object { "$((Get-FileHash -Algorithm SHA256 $_.FullName).Hash.ToLowerInvariant())  $($_.Name)" }
[IO.File]::WriteAllLines((Join-Path $dist 'SHA256SUMS.txt'),$checksumLines,[Text.UTF8Encoding]::new($false))
Remove-Item $setupPayload,$payload,$wixOut -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "Release-Artefakte erstellt: $dist"
Get-ChildItem $dist | Format-Table Name,Length
