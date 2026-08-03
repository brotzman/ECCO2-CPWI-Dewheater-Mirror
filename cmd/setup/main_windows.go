//go:build windows && installerpayload

package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	productName        = "ECCO2 CPWI Dew Mirror"
	version            = "1.0.1"
	uninstallKey       = `HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\ECCO2CPWIDewMirror`
	mbOK               = 0x00000000
	mbIconInfo         = 0x00000040
	mbIconError        = 0x00000010
	mbYesNo            = 0x00000004
	mbIconQuestion     = 0x00000020
	idYes              = 6
	errorAlreadyExists = 183
	createNoWindow     = 0x08000000
	detachedProcess    = 0x00000008
)

//go:embed payload/*
var payload embed.FS

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	shell32       = syscall.NewLazyDLL("shell32.dll")
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	messageBoxW   = user32.NewProc("MessageBoxW")
	isUserAnAdmin = shell32.NewProc("IsUserAnAdmin")
	createMutexW  = kernel32.NewProc("CreateMutexW")
	closeHandle   = kernel32.NewProc("CloseHandle")
)

type options struct {
	uninstall bool
	repair    bool
	quiet     bool
	logPath   string
}

func utf16Ptr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func message(text string, flags uintptr) int {
	r, _, _ := messageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(productName+" Setup"))), flags)
	return int(r)
}

func parseOptions(args []string) options {
	var o options
	for i := 0; i < len(args); i++ {
		a := strings.ToLower(strings.TrimSpace(args[i]))
		switch a {
		case "/uninstall", "-uninstall", "--uninstall":
			o.uninstall = true
		case "/repair", "-repair", "--repair":
			o.repair = true
		case "/quiet", "-quiet", "--quiet", "/silent":
			o.quiet = true
		case "/log", "-log", "--log":
			if i+1 < len(args) {
				i++
				o.logPath = args[i]
			}
		}
	}
	if o.logPath == "" {
		o.logPath = filepath.Join(os.TempDir(), "ECCO2-CPWI-Dew-Mirror-Setup.log")
	}
	return o
}

func logLine(path, text string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s  %s\r\n", time.Now().UTC().Format(time.RFC3339Nano), text)
}

func acquireMutex() (uintptr, error) {
	name := utf16Ptr(`Global\ECCO2CPWIDewMirrorSetup-65071386-3E88-46F8-9035-6A77E6839F22`)
	h, _, e := createMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return 0, e
	}
	if syscall.GetLastError() == syscall.Errno(errorAlreadyExists) {
		closeHandle.Call(h)
		return 0, fmt.Errorf("another setup instance is already running")
	}
	return h, nil
}

func admin() bool { r, _, _ := isUserAnAdmin.Call(); return r != 0 }

func quoteWindowsArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range s {
		if r == '\\' {
			slashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat("\\", slashes*2+1))
			b.WriteRune('"')
			slashes = 0
			continue
		}
		b.WriteString(strings.Repeat("\\", slashes))
		slashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat("\\", slashes*2))
	b.WriteByte('"')
	return b.String()
}

func relaunchElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		quoted = append(quoted, quoteWindowsArg(a))
	}
	argumentLine := strings.Join(quoted, " ")
	argumentClause := ""
	if argumentLine != "" {
		argumentClause = ` -ArgumentList ` + psQuote(argumentLine)
	}
	script := `$ErrorActionPreference='Stop'; $p=Start-Process -FilePath ` + psQuote(exe) + argumentClause + ` -Verb RunAs -Wait -PassThru; exit $p.ExitCode`
	return powershellEncoded(script, false)
}

func installDir() string {
	base := strings.TrimSpace(os.Getenv("ProgramFiles"))
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, productName)
}
func publicDesktop() string {
	if p := strings.TrimSpace(os.Getenv("PUBLIC")); p != "" {
		return filepath.Join(p, "Desktop")
	}
	return `C:\Users\Public\Desktop`
}
func startMenuDir() string {
	base := strings.TrimSpace(os.Getenv("ProgramData"))
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, `Microsoft\Windows\Start Menu\Programs`, productName)
}

func runHidden(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func powershellEncoded(script string, detached bool) error {
	u := utf16.Encode([]rune(script))
	raw := make([]byte, len(u)*2)
	for i, v := range u {
		raw[i*2] = byte(v)
		raw[i*2+1] = byte(v >> 8)
	}
	enc := base64.StdEncoding.EncodeToString(raw)
	c := exec.Command(filepath.Join(os.Getenv("SystemRoot"), `System32\WindowsPowerShell\v1.0\powershell.exe`), "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", enc)
	flags := uint32(createNoWindow)
	if detached {
		flags |= detachedProcess
	}
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags}
	if detached {
		return c.Start()
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PowerShell: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func createShortcut(path, target, working, description string) error {
	script := `$ErrorActionPreference='Stop'; $w=New-Object -ComObject WScript.Shell; $s=$w.CreateShortcut(` + psQuote(path) + `); $s.TargetPath=` + psQuote(target) + `; $s.WorkingDirectory=` + psQuote(working) + `; $s.Description=` + psQuote(description) + `; $s.Save()`
	return powershellEncoded(script, false)
}

func removePath(path string) { _ = os.RemoveAll(path) }

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func payloadFile(name string) ([]byte, error) { return payload.ReadFile("payload/" + name) }

func copySelf(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

func regAdd(name, typ, value string) error {
	return runHidden("reg.exe", "ADD", uninstallKey, "/v", name, "/t", typ, "/d", value, "/f")
}

func registerUninstall(dir string) error {
	uninstaller := filepath.Join(dir, "ECCO2CPWIDewMirrorUninstall.exe")
	if err := regAdd("DisplayName", "REG_SZ", productName); err != nil {
		return err
	}
	if err := regAdd("DisplayVersion", "REG_SZ", version); err != nil {
		return err
	}
	if err := regAdd("Publisher", "REG_SZ", "ECCO2 CPWI Dew Mirror Project"); err != nil {
		return err
	}
	if err := regAdd("InstallLocation", "REG_SZ", dir); err != nil {
		return err
	}
	if err := regAdd("DisplayIcon", "REG_SZ", filepath.Join(dir, "ECCO2CPWIDewMirror.exe")); err != nil {
		return err
	}
	if err := regAdd("UninstallString", "REG_SZ", `"`+uninstaller+`" /uninstall`); err != nil {
		return err
	}
	if err := regAdd("QuietUninstallString", "REG_SZ", `"`+uninstaller+`" /uninstall /quiet`); err != nil {
		return err
	}
	if err := regAdd("NoModify", "REG_DWORD", "1"); err != nil {
		return err
	}
	if err := regAdd("NoRepair", "REG_DWORD", "1"); err != nil {
		return err
	}
	return regAdd("EstimatedSize", "REG_DWORD", strconv.Itoa(12000))
}

func install(o options) error {
	dir := installDir()
	logLine(o.logPath, "install/repair to "+dir)
	files := []string{"ECCO2CPWIDewMirror.exe", "README.md", "README_DE.md", "CHANGELOG.md"}
	for _, name := range files {
		b, err := payloadFile(name)
		if err != nil {
			return err
		}
		if err = writeAtomic(filepath.Join(dir, name), b); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	uninstaller := filepath.Join(dir, "ECCO2CPWIDewMirrorUninstall.exe")
	if err := copySelf(uninstaller); err != nil {
		return fmt.Errorf("install uninstaller: %w", err)
	}
	if err := createShortcut(filepath.Join(publicDesktop(), productName+".lnk"), filepath.Join(dir, "ECCO2CPWIDewMirror.exe"), dir, productName+" "+version); err != nil {
		return err
	}
	menu := startMenuDir()
	if err := os.MkdirAll(menu, 0755); err != nil {
		return err
	}
	if err := createShortcut(filepath.Join(menu, productName+".lnk"), filepath.Join(dir, "ECCO2CPWIDewMirror.exe"), dir, productName+" "+version); err != nil {
		return err
	}
	if err := registerUninstall(dir); err != nil {
		return err
	}
	logLine(o.logPath, "installation completed")
	return nil
}

func scheduleSelfRemoval(dir string, pid int) error {
	script := `$ErrorActionPreference='SilentlyContinue'; for($i=0; $i -lt 300 -and (Get-Process -Id ` + strconv.Itoa(pid) + ` -ErrorAction SilentlyContinue); $i++){ Start-Sleep -Milliseconds 100 }; Start-Sleep -Milliseconds 500; Remove-Item -LiteralPath ` + psQuote(dir) + ` -Recurse -Force`
	return powershellEncoded(script, true)
}

func uninstall(o options) error {
	dir := installDir()
	logLine(o.logPath, "uninstall from "+dir)
	// Ensure the installed GUI cannot keep its executable or installation
	// directory locked while files are removed.
	_ = runHidden("taskkill.exe", "/IM", "ECCO2CPWIDewMirror.exe", "/T", "/F")
	removePath(filepath.Join(publicDesktop(), productName+".lnk"))
	removePath(startMenuDir())
	_ = runHidden("reg.exe", "DELETE", uninstallKey, "/f")
	for _, name := range []string{"ECCO2CPWIDewMirror.exe", "README.md", "README_DE.md", "CHANGELOG.md"} {
		_ = os.Remove(filepath.Join(dir, name))
	}
	if err := scheduleSelfRemoval(dir, os.Getpid()); err != nil {
		return err
	}
	logLine(o.logPath, "uninstall scheduled")
	return nil
}

func main() {
	o := parseOptions(os.Args[1:])
	// Acquire the machine-wide setup lock only in the elevated process. Holding
	// it in the unelevated parent would prevent the UAC child from starting.
	if !admin() {
		if err := relaunchElevated(os.Args[1:]); err != nil {
			logLine(o.logPath, "ERROR elevation: "+err.Error())
			if !o.quiet {
				message("Administratorrechte konnten nicht angefordert werden.\r\n\r\n"+err.Error(), mbOK|mbIconError)
			}
			os.Exit(1)
		}
		return
	}
	h, err := acquireMutex()
	if err != nil {
		logLine(o.logPath, "ERROR setup lock: "+err.Error())
		if !o.quiet {
			message(err.Error(), mbOK|mbIconError)
		}
		os.Exit(1)
	}
	defer closeHandle.Call(h)
	if o.uninstall && !o.quiet {
		if message("ECCO2 CPWI Dew Mirror wirklich deinstallieren?", mbYesNo|mbIconQuestion) != idYes {
			return
		}
	}
	if !o.uninstall && !o.repair && !o.quiet {
		if message(productName+" "+version+" installieren?", mbYesNo|mbIconQuestion) != idYes {
			return
		}
	}
	if o.uninstall {
		err = uninstall(o)
	} else {
		err = install(o)
	}
	if err != nil {
		logLine(o.logPath, "ERROR "+err.Error())
		if !o.quiet {
			message("Setup wurde mit einem Fehler beendet.\r\n\r\n"+err.Error()+"\r\n\r\nLog: "+o.logPath, mbOK|mbIconError)
		}
		os.Exit(1)
	}
	if !o.quiet {
		if o.uninstall {
			message("ECCO2 CPWI Dew Mirror wurde deinstalliert.", mbOK|mbIconInfo)
		} else {
			message("ECCO2 CPWI Dew Mirror wurde erfolgreich installiert.", mbOK|mbIconInfo)
		}
	}
}
