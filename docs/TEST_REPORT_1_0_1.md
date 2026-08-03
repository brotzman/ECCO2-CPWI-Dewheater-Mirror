# ECCO2 CPWI Dew Mirror 1.0.1 — Build/Test Report

Behobener Startfehler
---------------------
Der Win32-Hauptthread war nicht mit runtime.LockOSThread an einen festen
Windows-Thread gebunden. Fenstererstellung und GetMessage-Nachrichtenschleife
konnten dadurch auf verschiedenen OS-Threads laufen. Win32-Fenster und ihre
Message Queue sind jedoch an den erstellenden Thread gebunden. Version 1.0.1
bindet den vollständigen GUI-Lebenszyklus an einen OS-Thread.

Weitere Korrekturen
-------------------
- Statusfenster wird vor EAGLE-Netzwerkabfragen sichtbar gemacht.
- RegisterClassExW, CreateWindowExW, SetTimer und GetMessageW werden geprüft.
- Fehler werden per MessageBox und Startprotokoll sichtbar gemacht.
- Fehler beim Erstellen einzelner Steuerelemente werden nicht mehr ignoriert.
- Startprotokoll: %LOCALAPPDATA%\ECCO2CPWIDewMirror\Logs\startup.log
- Konfiguration: %LOCALAPPDATA%\ECCO2CPWIDewMirror\ECCO2CPWIDewMirror.json
- Alte Konfiguration neben der EXE wird einmalig übernommen.
- Statuszähler und Laufzustand wurden gegen konkurrierende Zugriffe abgesichert.

Automatisiert geprüft
---------------------
- Go unit and repository contract tests: 6/6 PASS
- Go race detector: PASS
- go vet: PASS
- Windows/amd64 vollständiger Compile-Test: PASS
- Windows/amd64 Test-Binary Compile: PASS
- Reproduzierbarer Doppelbuild: PASS, Anwendung und native Setup-EXE jeweils bytegenau identisch
- Anwendung und native Setup-EXE: PE32+ x86-64 Windows GUI
- ASLR / High-Entropy-ASLR / DEP-NX: vorhanden
- Startup source contract: PASS
- Installer-/Workflow-Verträge: PASS
- Setup-Elevation erfolgt vor dem globalen Setup-Mutex: PASS
- Setup-Fehler liefern einen von null verschiedenen Exitcode: PASS
- Statusfensterfehler werden nicht mehr still beendet: PASS
- EAGLE-Polling startet erst nach WINDOW_VISIBLE: PASS
- Quellarchiv ohne Cache-, Objekt- oder Temporärdateien: PASS

Manueller Windows-Test
----------------------
Test-GuiStartup.ps1 startet die EXE, wartet auf das Statusfenster, prüft
Fenstertitel und Reaktionsfähigkeit und schließt das Fenster anschließend
kontrolliert.

In dieser Linux-Prüfumgebung nicht direkt ausführbar
----------------------------------------------------
- Interaktiver Win32-GUI-Smoke-Test
- CPWI-Erkennung an einem echten Windows-/CPWI-System
- Virtual-COM-Treiberverhalten
- EAGLE2/ECCO2-Live-Telemetrie

Sicherheitsinvariante
---------------------
Die Anwendung enthält keinen EAGLE2-Set-Endpunkt. CPWI-Schreibopcodes werden
nur beantwortet/protokolliert und verändern keine reale Heizleistung.
