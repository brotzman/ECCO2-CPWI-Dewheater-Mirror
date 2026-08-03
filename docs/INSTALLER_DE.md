# Installer

Das offizielle Installationsprogramm ist `ECCO2-CPWI-Dew-Mirror-1.0.1-Setup.exe`.

Es installiert die Anwendung pro Maschine unter:

```text
C:\Program Files\ECCO2 CPWI Dew Mirror
```

Angelegt werden eine Desktopverknüpfung und ein Startmenüeintrag. Benutzereinstellungen und Logs liegen unter `%LOCALAPPDATA%\ECCO2CPWIDewMirror` und werden bei einer Deinstallation nicht gelöscht.

Der Installer enthält keinen virtuellen COM-Treiber.

Der eigenständige Setup-Launcher ist in Go implementiert und kann ohne WiX gebaut werden. GitHub Actions erzeugt zusätzlich ein WiX-MSI.
