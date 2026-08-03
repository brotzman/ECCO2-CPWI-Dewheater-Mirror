#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags '-s -w -H=windowsgui' -o dist/ECCO2CPWIDewMirror.exe .
file dist/ECCO2CPWIDewMirror.exe
