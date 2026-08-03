[CmdletBinding()]
param(
    [string]$Exe = (Join-Path $PSScriptRoot 'ECCO2CPWIDewMirror.exe'),
    [int]$TimeoutSeconds = 15,
    [string]$DiagnosticDirectory = ''
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Exe -PathType Leaf)) { throw "EXE fehlt: $Exe" }
if ([string]::IsNullOrWhiteSpace($DiagnosticDirectory)) {
    $DiagnosticDirectory = Join-Path $env:TEMP 'ECCO2-CPWI-Dew-Mirror-Gui-Test'
}
New-Item -ItemType Directory -Force -Path $DiagnosticDirectory | Out-Null
$startupLog = Join-Path $env:LOCALAPPDATA 'ECCO2CPWIDewMirror\Logs\startup.log'
$copiedStartupLog = Join-Path $DiagnosticDirectory 'gui-startup.log'
Remove-Item -LiteralPath $startupLog -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $copiedStartupLog -Force -ErrorAction SilentlyContinue

function Get-StartupLogText {
    if (Test-Path -LiteralPath $startupLog -PathType Leaf) {
        return (Get-Content -LiteralPath $startupLog -Raw -ErrorAction SilentlyContinue)
    }
    return ''
}
function Copy-StartupDiagnostics {
    if (Test-Path -LiteralPath $startupLog -PathType Leaf) {
        Copy-Item -LiteralPath $startupLog -Destination $copiedStartupLog -Force -ErrorAction SilentlyContinue
    }
}
function Save-WindowScreenshot([IntPtr]$Handle) {
    if ($Handle -eq [IntPtr]::Zero) { return }
    try {
        if (-not ('ECCO2.NativeWindow' -as [type])) {
            Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
namespace ECCO2 {
    public static class NativeWindow {
        [StructLayout(LayoutKind.Sequential)]
        public struct RECT { public int Left, Top, Right, Bottom; }
        [DllImport("user32.dll")]
        public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
    }
}
'@
        }
        Add-Type -AssemblyName System.Drawing
        $rect = [ECCO2.NativeWindow+RECT]::new()
        if (-not [ECCO2.NativeWindow]::GetWindowRect($Handle, [ref]$rect)) { return }
        $width = $rect.Right - $rect.Left
        $height = $rect.Bottom - $rect.Top
        if ($width -le 0 -or $height -le 0) { return }
        $bitmap = [Drawing.Bitmap]::new($width, $height)
        $graphics = [Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphics.CopyFromScreen($rect.Left, $rect.Top, 0, 0, $bitmap.Size)
            $bitmap.Save((Join-Path $DiagnosticDirectory 'gui-window.png'), [Drawing.Imaging.ImageFormat]::Png)
        } finally {
            $graphics.Dispose()
            $bitmap.Dispose()
        }
    } catch {
        Write-Host "DIAGNOSTIC WARNING: Screenshot konnte nicht gespeichert werden: $($_.Exception.Message)"
    }
}

$p = Start-Process -FilePath $Exe -PassThru
try {
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    $windowVisibleMarker = $false
    do {
        Start-Sleep -Milliseconds 200
        $p.Refresh()
        $startupText = Get-StartupLogText
        $windowVisibleMarker = $startupText -match 'WINDOW_VISIBLE'
        if ($p.HasExited) {
            Copy-StartupDiagnostics
            throw "Programm wurde vor dem stabilen Statusfenster beendet (Exitcode $($p.ExitCode)). Startup-Log: $startupText"
        }
    } while ($p.MainWindowHandle -eq 0 -and -not $windowVisibleMarker -and [DateTime]::UtcNow -lt $deadline)

    Copy-StartupDiagnostics
    if ($p.MainWindowHandle -eq 0 -and -not $windowVisibleMarker) {
        throw "Innerhalb von $TimeoutSeconds Sekunden wurde weder ein Statusfenster noch der Marker WINDOW_VISIBLE erkannt."
    }

    if ($p.MainWindowHandle -ne 0) {
        if ($p.MainWindowTitle -notlike 'ECCO2 CPWI Dew Mirror*') { throw "Unerwarteter Fenstertitel: $($p.MainWindowTitle)" }
        if (-not $p.Responding) { throw 'Das Statusfenster reagiert nicht.' }
        Save-WindowScreenshot ([IntPtr]$p.MainWindowHandle)
        Write-Host "PASS: Statusfenster geöffnet: $($p.MainWindowTitle)"
        if (-not $p.CloseMainWindow()) { throw 'WM_CLOSE konnte nicht gesendet werden.' }
        if (-not $p.WaitForExit(5000)) { throw 'Programm beendet sich nach dem Schließen nicht.' }
        if ($p.ExitCode -ne 0) { throw "Programm meldet nach normalem Schließen Exitcode $($p.ExitCode)." }
        Write-Host 'PASS: Statusfenster sauber geschlossen.'
    } else {
        # Some hosted Windows sessions do not expose MainWindowHandle even
        # though USER32 created and showed the window. The application-owned
        # startup marker plus a still-running process is the reliable fallback.
        Write-Host 'PASS: WINDOW_VISIBLE im Startup-Log bestätigt; der CI-Desktop stellte keinen MainWindowHandle bereit.'
    }
} catch {
    Copy-StartupDiagnostics
    Write-Host "FAIL: GUI-Smoke-Test: $($_.Exception.Message)"
    $startupText = Get-StartupLogText
    if (-not [string]::IsNullOrWhiteSpace($startupText)) {
        Write-Host '--- startup.log ---'
        Write-Host $startupText
        Write-Host '--- end startup.log ---'
    }
    throw
} finally {
    if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
}
