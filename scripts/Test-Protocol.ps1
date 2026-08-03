# Test der 2X-Emulation über die Gegenseite des virtuellen COM-Paares.
# Beispiel: Mirror = COM20, dieser Test = COM21.
$port = "COM21"
$baud = 115200

$sp = New-Object System.IO.Ports.SerialPort $port,$baud,'None',8,'One'
$sp.ReadTimeout = 1500
$sp.WriteTimeout = 1500
$sp.Open()
try {
    # GET_VERSION an Device 0xBB: 3B 03 0D BB FE 37
    [byte[]]$req = 0x3B,0x03,0x0D,0xBB,0xFE,0x37
    $sp.Write($req,0,$req.Length)
    Start-Sleep -Milliseconds 100
    $buf = New-Object byte[] 64
    $n = $sp.Read($buf,0,$buf.Length)
    $hex = ($buf[0..($n-1)] | ForEach-Object { $_.ToString('X2') }) -join ' '
    Write-Host "Antwort: $hex"
    Write-Host "Erwarteter Beginn: 3B 07 BB 0D FE 01 01 04 F6 37"
} finally {
    $sp.Close()
}
