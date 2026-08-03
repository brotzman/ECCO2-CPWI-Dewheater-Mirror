# ECCO2 CPWI Dew Mirror 1.0.1 — Build/Test Report

## Ursprünglicher Buildfehler

Der erste bereitgestellte GitHub-Actions-Lauf brach bei `go vet` ab. Ursache war
eine direkte `uintptr`→`unsafe.Pointer`-Konvertierung im Win32-Callback für
`WM_GETMINMAXINFO`. Diese Stelle wurde über `RtlMoveMemory` abgesichert und durch
einen Regressionstest geschützt.

## Ergebnis des nachfolgenden Windows-/MSI-Laufs vom 3. August 2026

Die bereitgestellten MSI-Protokolle bestätigen auf einem sauberen Windows-Runner:

- MSI-Installation: **PASS**, Windows-Installer-Status `0`
- GUI-Smoke-Test nach Installation: durch das Testskript vorausgesetzt und bestanden
- erzwungene MSI-Reparatur nach Löschen von `ECCO2CPWIDewMirror.exe`: **PASS**
- Wiederherstellung der EXE mit identischem SHA-256-Hash: **PASS**
- MSI-Deinstallation: **PASS**, Windows-Installer-Status `0`
- Entfernung des Installationsverzeichnisses: **PASS**

Die MSI-Loghinweise `1728` (Konfiguration abgeschlossen) und `1724`
(Entfernen abgeschlossen) sind Erfolgsmeldungen. Es gibt kein `Return value 3`,
keinen Status ungleich `0` und keinen Rückgabecode ungleich `0`. Der Hinweis auf
eine fehlende `MsiPatchCertificate`-Tabelle betrifft ausschließlich LUA-Patching
und ist für dieses ungepatchte MSI unkritisch. Die Artefakte sind derzeit nicht
digital signiert.

## Zusätzlich geprüfte Release-Artefakte

- Alle Einträge in `SHA256SUMS.txt` stimmen mit den gelieferten Dateien überein.
- Portable- und Quell-ZIP sind strukturell fehlerfrei.
- Anwendung und native Setup-EXE sind Windows-x64-GUI-Binaries.
- Das MSI ist x64/de-DE, Version 1.0.1, erstellt mit WiX 5.0.2.
- ProductCode: `{68E505D6-0CB5-407E-98A1-11D043404CB9}`
- UpgradeCode: `{9F8E1513-812C-4D29-9432-C14B43AB6384}`

## Bei der Nachprüfung erkannte Repository-Verbesserungen

Das erzeugte Quellarchiv enthielt versehentlich `artifacts/test-logs` und ließ
die Dotfiles `.gitignore`, `.gitattributes` und `.editorconfig` aus. Außerdem
behauptete die englische README noch, es sei keine Lizenz vorhanden, obwohl
`LICENSE` die GPL Version 3 enthält.
Die bisherigen Portable- und Installer-Payloads enthielten den Lizenztext ebenfalls nicht.

Korrigiert wurden deshalb:

- Quellpaket schließt `artifacts`, `ci-input`, `dist`, `.git`, `.config`, `.wix`
  und generierte Installer-Verzeichnisse aus.
- Quellpaket wird mit `ZipFile.CreateFromDirectory` erstellt und enthält dadurch
  auch Repository-Dotfiles und `LICENSE`.
- neuer `Test-ReleaseArtifacts.ps1` prüft Artefakte, SHA-256-Werte und ZIP-Inhalte.
- Installer-Tests prüfen nun Desktop- und Startmenü-Verknüpfung sowie
  Deinstallationsregistrierung vor und nach der Deinstallation.
- der native Setup-Reparaturtest löscht die EXE vor der Reparatur, genau wie der
  MSI-Test.
- GitHub Actions archiviert zusätzlich eine lesbare Zusammenfassung jedes
  Setup-/MSI-Testlaufs.
- Lizenzhinweise in README und README_DE entsprechen jetzt GPL-3.0.
- Portable-ZIP, native Setup-EXE und MSI enthalten die Datei `LICENSE`; die
  Installer-Tests prüfen deren Installation.

## Lokal erneut geprüft

Mit Go 1.23.2 unter Linux:

- `gofmt`
- `go test -count=1 ./...`
- `go test -race -count=1 ./...`
- `go vet ./...`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...`
- Windows-x64-GUI-Cross-Build der Anwendung
- Windows-x64-GUI-Cross-Build des nativen Setup-Bootstrappers
- XML-Prüfung des WiX-Manifests
- YAML-Prüfung des GitHub-Actions-Workflows
- Archivsauberkeitsprüfung des Repository-ZIP

## Noch ausstehend

Die **erweiterten** Setup- und MSI-Tests sowie der korrigierte Quellpaket-Test
müssen einmal in einem neuen GitHub-Actions-Lauf ausgeführt werden. Für den
nativen Setup-Lauf wurden in diesem Upload keine Setup-Testlogs bereitgestellt;
daher wird dessen bisheriger Erfolg nicht aus den MSI-Logs abgeleitet.

## Sicherheitsinvariante

Die Anwendung verwendet im Go-Quellcode ausschließlich HTTP-GET-Zugriffe auf
`/getsupply`, `/getecco` und `/getregout`. Es existiert kein aufrufbarer
EAGLE2-Schreibendpunkt. CPWI-Schreibopcodes werden weiterhin nur beantwortet
und protokolliert; die reale ECCO2-Heizregelung bleibt unverändert.

## Installer-Testhärtung vom 03.08.2026

- Registry-Ergebnisse mit null, einem oder mehreren Treffern werden unabhängig von PowerShell-Pipeline-Unwrapping sicher gezählt.
- Setup-Deinstallation wird als asynchroner Selbstlöschvorgang behandelt und bis zu 45 Sekunden nachkontrolliert.
- Der native Uninstaller wiederholt das Entfernen des Installationsverzeichnisses, falls kurzlebige Dateihandles den ersten Versuch blockieren.
