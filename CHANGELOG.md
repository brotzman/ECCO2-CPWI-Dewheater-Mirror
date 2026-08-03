# ECCO2 CPWI Dew Mirror 1.0.1 — 03.08.2026

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
