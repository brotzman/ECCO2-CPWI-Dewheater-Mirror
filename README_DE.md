# ECCO2 CPWI Dew Mirror 1.0.1

Zweck
-----
Die Anwendung spiegelt die vom PrimaLuceLab EAGLE2 Manager bereitgestellte
ECCO2-Telemetrie gegenüber Celestron CPWI als Smart DewHeater Controller 2X.

WICHTIG: Version 1.0.1 ist absichtlich READ ONLY.
- Die EAGLE2-API wird nur mit GET-Anfragen gelesen.
- CPWI-Schreibbefehle (Heizerleistung, Auto-Aggressivität, Recalibration,
  Enable/Disable) werden protokolliert und quittiert, aber NICHT an EAGLE2/ECCO2
  weitergereicht.
- Es gibt im Programm keinen Aufruf von /setregout.
- ECCO2/EAGLE2 bleibt alleiniger Regler der echten Tauheizung.

Datenzuordnung
--------------
CPWI / 2X                     EAGLE2 / ECCO2
Ambient Temperature           /getecco temp
Humidity                      /getecco hum
Dew Point                     /getecco dew
Thermistor / Heater 1         /getecco temp5 (T5)
Thermistor / Heater 2         /getecco temp6 (T6)
Heater 1 output               /getregout?idx=1, aus Spannung/Supply abgeleitet
Heater 2 output               /getregout?idx=2, aus Spannung/Supply abgeleitet
Input voltage                 /getsupply

T7/Port 7 wird nicht emuliert, weil ein echter Celestron 2X zwei Heizkanäle hat.

Voraussetzungen
---------------
1. Eagle2Manager.exe muss laufen und seine lokale API auf 127.0.0.1:1380
   bereitstellen.
2. ECCO2 muss im EAGLE2 Manager verbunden sein.
3. Für die CPWI-Seite wird ein virtuelles Nullmodem-/COM-Paar benötigt.
   Beispiel: COM20 <-> COM21. Dieses Programm öffnet COM20, CPWI bzw. der
   Testclient öffnet COM21.
   Ein Virtual-COM-Treiber wird aus Sicherheits-/Treibergründen NICHT gebündelt.
4. CPWI 2.4+ / 2.5.x ist der Zielstand der Emulation.

Schnelltest ohne CPWI
---------------------
- Virtuelles COM-Paar anlegen, z.B. COM20 <-> COM21.
- Im Mirror "Mirror COM" = COM20 eintragen.
- "Mirror starten".
- Test-Protocol.ps1 bearbeiten und $port="COM21" setzen.
- PowerShell ausführen. Erwartet wird eine 2X-GET_VERSION-Antwort.

CPWI-Test
---------
Die Celestron-2X-Kommunikation verwendet AUX-Pakete. Ob CPWI einen separaten
virtuellen Port direkt als Controller-Verbindung akzeptiert, hängt vom in CPWI
verwendeten Verbindungspfad ab. Die App besitzt deshalb zusätzlich ein
optionales Feld "AUX-Passthrough". Dies ist für einen echten Celestron-PC/AUX-
Transport gedacht und standardmäßig LEER.

Wenn CPWI das Gerät noch nicht erkennt:
- Den Protokollbereich der App vollständig kopieren.
- In CPWI "Detailed logging" einschalten.
- CPWI-Verbindungslog zusammen mit dem Mirror-Protokoll sichern.
Damit können unbekannte Discovery-Kommandos in einer Folgerevision ergänzt
werden, ohne die reale ECCO2-Regelung anzufassen.

Protokollbasis
--------------
Implementiert sind die öffentlich dokumentierten/reverse-engineerten 2X-AUX-
Kommandos für Device 0xBB (Version/Boot) und 0x17 (Hauptgerät):
GET_VERSION (0xFE), INPUT_POWER (0x00), INPUT_LIMITS (0x04), NUM_PORTS (0x10),
QUERY_PORT (0x11), QUERY_HEATER (0x12), ENABLE_PORT (0x14), SET_AUTO_AGGR (0x16),
SET_MANUAL_PWM (0x17), QUERY_ENVIRONMENT (0x18), RECALIBRATE (0x19) und
QUERY_CALIBRATED (0x1A).

Schreibkommandos werden ausschließlich leer quittiert und ignoriert.

Build
-----
Windows x64:
  build-windows-x64.bat

Cross-Build mit Go:
  ./build-cross.sh

Keine Drittanbieter-Go-Pakete erforderlich.

Statusfenster und Startdiagnose
-------------------------------
Version 1.0.1 bindet das Win32-Fenster und seine Nachrichtenschleife fest an
EINEN Windows-Thread. Das Statusfenster wird vor den ersten EAGLE-Abfragen
sichtbar gemacht.

Falls das Fenster dennoch nicht erscheint, wird ein sichtbarer Fehlerdialog
angezeigt. Das Startprotokoll befindet sich unter:
  %LOCALAPPDATA%\ECCO2CPWIDewMirror\Logs\startup.log

Die Konfiguration wird pro Benutzer gespeichert unter:
  %LOCALAPPDATA%\ECCO2CPWIDewMirror\ECCO2CPWIDewMirror.json

Windows-GUI-Smoke-Test:
  powershell -ExecutionPolicy Bypass -File .\scripts\Test-GuiStartup.ps1 -Exe .\dist\ECCO2CPWIDewMirror.exe


GitHub-/Installer-Build
-----------------------
Vollständige Artefakte mit Setup-EXE und MSI:
  .\scripts\Build-Release.ps1

Ohne WiX/MSI:
  .\scripts\Build-Release.ps1 -SkipMsi

Setup-EXE und MSI sind alternative Installationswege und sollen nicht parallel installiert werden.
