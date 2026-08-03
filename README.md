# ECCO2 CPWI Dew Mirror 1.0.1

Windows-x64 bridge that mirrors PrimaLuceLab EAGLE2/ECCO2 telemetry to Celestron CPWI as a Smart DewHeater Controller 2X.

**Safety model:** the application is deliberately read-only. CPWI write commands are acknowledged and logged but never forwarded to EAGLE2/ECCO2. The real dew-heater controller remains authoritative.

German documentation: [README_DE.md](README_DE.md)

## Build

Full build with native Setup EXE and MSI (requires Go, .NET SDK, and WiX 5):

```powershell
.\scripts\Build-Release.ps1
```

Build without MSI/WiX:

```powershell
.\scripts\Build-Release.ps1 -SkipMsi
```

The GitHub Actions workflow builds and tests the application on Windows, creates a native self-contained Setup EXE, an MSI, a portable ZIP, checksums, and optional GitHub Releases.

## Requirements at runtime

- Windows x64
- EAGLE2 Manager API at `127.0.0.1:1380`
- A virtual null-modem COM pair for CPWI communication

The repository does not bundle a virtual COM driver.

## License

No licence file was present in the supplied source. Add the intended licence before publishing the repository publicly.

Do not install the native Setup EXE and raw MSI side by side; they are alternative distribution formats.
