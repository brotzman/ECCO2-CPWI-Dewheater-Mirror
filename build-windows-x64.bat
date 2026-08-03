@echo off
setlocal
cd /d "%~dp0"
echo Testing ECCO2 CPWI Dew Mirror 1.0.1...
go test ./...
if errorlevel 1 exit /b 1
go vet ./...
if errorlevel 1 exit /b 1
if not exist dist mkdir dist
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -buildvcs=false -ldflags "-s -w -H=windowsgui" -o dist\ECCO2CPWIDewMirror.exe .
if errorlevel 1 exit /b 1
echo Built dist\ECCO2CPWIDewMirror.exe
