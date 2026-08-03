# ECCO2 CPWI Dew Mirror 1.0.1 — 03.08.2026

- Weiteren PowerShell-`StrictMode`-Fehler behoben: Einzelne Registry-Treffer werden nicht mehr über eine unsichere `.Count`-Eigenschaft ausgewertet.
- Installer-Test verwendet nun eine feste Ganzzahlfunktion für die Anzahl passender Deinstallationsregistrierungen.
- Nach Setup- und MSI-Deinstallation wird bis zu 45 Sekunden kontrolliert auf die vollständige Entfernung gewartet, statt nur zwei Sekunden zu pausieren.
- Die native Setup-Deinstallation versucht die selbstlöschende Installationsmappe mehrfach zu entfernen und toleriert kurzlebige Explorer-, Antivirus- oder Dateisystemhandles.
- Installer-Test korrigiert: Registry-Einträge ohne `DisplayName` oder `DisplayVersion` lösen unter PowerShell `StrictMode` keinen Abbruch mehr aus.
- Die Produkterkennung liest Registry-Eigenschaften jetzt defensiv über `PSObject.Properties`.
- Dadurch können Setup- und MSI-Tests nach erfolgreicher Installation bis zu GUI-Start, Reparatur und Deinstallation fortgesetzt werden.
- Neues App-Icon integriert und in Portable-ZIP, Setup und MSI ausgeliefert.
- Desktop- und Startmenü-Verknüpfungen verwenden jetzt automatisch das neue Icon.
- Installer und MSI hinterlegen das neue Produkticon auch in der Windows-Softwareliste.
- Statusfenster lädt das Icon ebenfalls zur Laufzeit für Titelleiste und Taskleiste.
- Abstände und Höhen im Hauptfenster wurden überarbeitet, um Clipping im Bereich „ECCO2 Telemetrie“ zu vermeiden.
- Minimale Fenstergröße erhöht, damit die überarbeitete Anordnung stabil und lesbar bleibt.
- MSI-Installation, erzwungene Reparatur und Deinstallation wurden auf einem sauberen Windows-Runner erfolgreich bestätigt.
- Installer-Tests prüfen jetzt zusätzlich Startmenü-Verknüpfung, Deinstallationsregistrierung und deren vollständige Entfernung.
- Auch der native Setup-Reparaturtest löscht die Programmdatei vorab und muss sie wiederherstellen.
- Das Quellarchiv enthält nun die Repository-Dotfiles und `LICENSE`, aber keine Build-Logs oder generierten Verzeichnisse.
- Ein neuer Release-Artefakttest validiert Prüfsummen sowie Portable- und Quellarchive.
- README-Lizenzhinweise wurden an die vorhandene GPL-3.0-Lizenz angepasst.
- Portable-ZIP, native Setup-EXE und MSI enthalten nun ebenfalls `LICENSE`.
- Korrigiert den Windows-CI-Abbruch durch `go vet` bei `WM_GETMINMAXINFO`.
- Die Win32-Struktur wird ohne direkte `uintptr`→`unsafe.Pointer`-Konvertierung gelesen und zurückgeschrieben.
- GitHub-Actions-PowerShell-Schritte verwenden jetzt PowerShell 7; Build-Logs werden dadurch als UTF-8 erzeugt.
- Regressionstest verhindert die erneute Einführung der unsicheren Konvertierung.
- Behebt den fehlenden beziehungsweise unzuverlässigen Start des Statusfensters.
- Win32-Fenster und Nachrichtenschleife bleiben jetzt sicher auf demselben OS-Thread.
- Das Statusfenster wird vollständig aufgebaut und sichtbar gemacht, bevor EAGLE-Abfragen starten.
- Fehler beim Registrieren der Fensterklasse, Erstellen des Fensters, der Steuerelemente,
  des Timers oder der Nachrichtenschleife werden sichtbar gemeldet.
- Startdiagnose unter %LOCALAPPDATA%\ECCO2CPWIDewMirror\Logs\startup.log.
- Konfiguration wird pro Benutzer statt neben der EXE gespeichert.
- Alte Konfiguration neben der EXE wird einmalig übernommen.
- Statuszähler und Laufzustand wurden gegen konkurrierende Zugriffe abgesichert.
- Neuer Windows-GUI-Smoke-Test Test-GuiStartup.ps1.

ECCO2 CPWI Dew Mirror 1.0.0 - 31.07.2026

- Erste Windows-x64-Version.
- ECCO2-Telemetrie über die lokale EAGLE2-Manager-API.
- Read-only-Emulation eines Celestron Smart DewHeater Controller 2X.
- AUX-Geräteadressen 0xBB und 0x17.
- 2X-Umgebung, Taupunkt, Feuchte, T5/T6 und zwei virtuelle Heizkanäle.
- CPWI-Schreibbefehle werden protokolliert, aber niemals an EAGLE/ECCO2 gesendet.
- Virtueller COM-Port als CPWI-Endpunkt.
- Optionaler AUX-Passthrough für geeignete Celestron-PC/AUX-Transporte.
- Live-Protokoll und Zähler für RX/TX, ignorierte Writes und unbekannte Opcodes.
