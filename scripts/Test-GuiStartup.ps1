param(
    [string]$Exe = (Join-Path $PSScriptRoot 'ECCO2CPWIDewMirror.exe'),
    [int]$TimeoutSeconds = 15
)
$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Exe -PathType Leaf)) { throw "EXE fehlt: $Exe" }
$p = Start-Process -FilePath $Exe -PassThru
try {
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        Start-Sleep -Milliseconds 200
        $p.Refresh()
        if ($p.HasExited) { throw "Programm wurde vor dem Statusfenster beendet (Exitcode $($p.ExitCode))." }
    } while ($p.MainWindowHandle -eq 0 -and [DateTime]::UtcNow -lt $deadline)
    if ($p.MainWindowHandle -eq 0) { throw "Innerhalb von $TimeoutSeconds Sekunden wurde kein Statusfenster geöffnet." }
    if ($p.MainWindowTitle -notlike 'ECCO2 CPWI Dew Mirror*') { throw "Unerwarteter Fenstertitel: $($p.MainWindowTitle)" }
    if (-not $p.Responding) { throw 'Das Statusfenster reagiert nicht.' }
    Write-Host "PASS: Statusfenster geöffnet: $($p.MainWindowTitle)"
    if (-not $p.CloseMainWindow()) { throw 'WM_CLOSE konnte nicht gesendet werden.' }
    if (-not $p.WaitForExit(5000)) { throw 'Programm beendet sich nach dem Schließen nicht.' }
    if ($p.ExitCode -ne 0) { throw "Programm meldet nach normalem Schließen Exitcode $($p.ExitCode)." }
    Write-Host 'PASS: Statusfenster sauber geschlossen.'
}
finally {
    if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
}
