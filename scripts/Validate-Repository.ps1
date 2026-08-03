Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$required=@('main_windows.go','protocol.go','version.go','go.mod','installer\wix\Package.wxs','installer\build-wix.ps1','cmd\setup\main_windows.go','scripts\Build-Release.ps1','scripts\Test-Installer.ps1','.github\workflows\build-release.yml')
$missing=$required | Where-Object { -not (Test-Path (Join-Path $root $_) -PathType Leaf) }
if ($missing) { throw "Missing repository files: $($missing -join ', ')" }
$generated=Get-ChildItem $root -Recurse -Force -File | Where-Object { $_.Extension -in @('.exe','.msi','.pdb','.pyc') -and $_.FullName -notlike "*\dist\*" }
if ($generated) { throw "Generated files are committed: $($generated.FullName -join ', ')" }
Write-Host 'PASS: repository structure is complete and clean.'
