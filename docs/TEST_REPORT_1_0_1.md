# ECCO2 CPWI Dew Mirror 1.0.1 — Build/Test Report

## Auswertung des fehlgeschlagenen GitHub-Actions-Laufs

Die bereitgestellten Logs zeigen:

- Repository-Validierung: **PASS**
- Go-Unit-Tests: **PASS**
- Abbruch in `scripts/Build-Release.ps1` bei `go vet`
- Ursache: `main_windows.go` wandelte den von Win32 gelieferten `lParam` in
  `WM_GETMINMAXINFO` direkt von `uintptr` in `unsafe.Pointer` um. Der
  `unsafeptr`-Analyzer meldete deshalb `possible misuse of unsafe.Pointer`.
- Der Lauf erreichte Anwendungsbuild, Setup-Build, WiX-MSI-Build und
  Installer-Tests nicht. Aus diesem fehlgeschlagenen Lauf lässt sich daher
  kein Ergebnis für diese nachgelagerten Stufen ableiten.

## Korrektur

- `WM_GETMINMAXINFO` verwendet keine direkte `uintptr`→`unsafe.Pointer`-
  Konvertierung mehr. Die Win32-Struktur wird innerhalb des Callbacks über
  `RtlMoveMemory` in eine lokale Go-Struktur kopiert, angepasst und wieder
  zurückgeschrieben.
- Ein Regressionstest schlägt fehl, falls das beanstandete Muster erneut
  eingeführt wird.
- Die Windows-Schritte im GitHub-Actions-Workflow verwenden PowerShell 7
  (`pwsh`), sodass die Testlogs als UTF-8 statt als UTF-16 ausgegeben werden.
- Unbenutzte Win32-Konstanten, Prozeduren, Handles und der nicht vorhandene
  Refresh-Befehl wurden entfernt.
- Setup- und WiX-Beschreibungen beziehen ihre Versionsangabe nun aus den
  vorhandenen Versionsvariablen, um Versionsdrift zu vermeiden.

## Lokal automatisiert geprüft

Mit Go 1.23.2 unter Linux wurden erfolgreich ausgeführt:

- `gofmt`-Prüfung
- `go test -count=1 ./...`
- `go test -race -count=1 ./...`
- `go vet ./...`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...`
- Windows-x64-GUI-Cross-Build der Anwendung
- Kompilierung des Windows-Testprogramms
- Windows-x64-GUI-Cross-Build des nativen Setup-Bootstrappers
- Bash-Syntaxprüfung von `build-cross.sh`
- XML-Prüfung von `installer/wix/Package.wxs`
- YAML-Prüfung von `.github/workflows/build-release.yml`
- Prüfung auf versehentlich enthaltene EXE-, MSI-, PDB- und Cache-Dateien

## Noch auf GitHub Actions beziehungsweise Windows zu prüfen

Ein neuer Workflow-Lauf ist weiterhin erforderlich für:

- Build mit der im Workflow festgelegten Go-Version 1.26.5
- WiX-5-MSI-Erstellung
- Installation, Reparatur und Deinstallation der nativen Setup-EXE
- Installation, Reparatur und Deinstallation des MSI
- Win32-GUI-Smoke-Test auf einem echten Windows-Runner

## Sicherheitsinvariante

Die Anwendung verwendet im Go-Quellcode ausschließlich HTTP-GET-Zugriffe auf
`/getsupply`, `/getecco` und `/getregout`. Es existiert kein aufrufbarer
EAGLE2-Schreibendpunkt. CPWI-Schreibopcodes werden weiterhin nur beantwortet
und protokolliert; die reale ECCO2-Heizregelung bleibt unverändert.
