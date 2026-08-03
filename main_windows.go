//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	appTitle  = appName + " " + appVersion
	className = "ECCO2CPWIDewMirrorWindow"

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	ES_READONLY         = 0x0800
	ES_AUTOHSCROLL      = 0x0080
	ES_MULTILINE        = 0x0004
	ES_AUTOVSCROLL      = 0x0040
	BS_PUSHBUTTON       = 0
	BS_GROUPBOX         = 7
	SS_LEFT             = 0
	SS_CENTERIMAGE      = 0x200
	SW_SHOWNORMAL       = 1
	CW_USEDEFAULT       = 0x80000000
	IDC_ARROW           = 32512
	IDI_APPLICATION     = 32512
	MB_OK               = 0x00000000
	MB_ICONERROR        = 0x00000010
	MB_SETFOREGROUND    = 0x00010000
	WM_DESTROY          = 0x0002
	WM_CLOSE            = 0x0010
	WM_SIZE             = 0x0005
	WM_COMMAND          = 0x0111
	WM_TIMER            = 0x0113
	WM_SETFONT          = 0x0030
	WM_CTLCOLORSTATIC   = 0x0138
	WM_CTLCOLOREDIT     = 0x0133
	WM_GETMINMAXINFO    = 0x0024
	WM_SETICON          = 0x0080
	SIZE_MINIMIZED      = 1
	TRANSPARENT         = 1
	IMAGE_ICON          = 1
	LR_LOADFROMFILE     = 0x0010
	LR_DEFAULTSIZE      = 0x0040
	ICON_SMALL          = 0
	ICON_BIG            = 1
	FW_NORMAL           = 400
	FW_SEMIBOLD         = 600
	CLEARTYPE_QUALITY   = 5
	DEFAULT_CHARSET     = 1
	OUT_DEFAULT_PRECIS  = 0
	CLIP_DEFAULT_PRECIS = 0
	DEFAULT_PITCH       = 0
	FF_DONTCARE         = 0

	ID_EAGLE = 1001
	ID_VCOM  = 1002
	ID_UPCOM = 1003
	ID_START = 1004
	ID_SAVE  = 1005
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	LPrivate       uint32
}
type WNDCLASSEXW struct {
	CbSize                                   uint32
	Style                                    uint32
	LpfnWndProc                              uintptr
	CbClsExtra                               int32
	CbWndExtra                               int32
	HInstance, HIcon, HCursor, HbrBackground uintptr
	LpszMenuName, LpszClassName              *uint16
	HIconSm                                  uintptr
}
type MINMAXINFO struct{ PtReserved, PtMaxSize, PtMaxPosition, PtMinTrackSize, PtMaxTrackSize POINT }
type DCB struct {
	DCBlength, BaudRate, Flags                     uint32
	WReserved, XonLim, XoffLim                     uint16
	ByteSize, Parity, StopBits                     byte
	XonChar, XoffChar, ErrorChar, EofChar, EvtChar byte
	WReserved1                                     uint16
}
type COMMTIMEOUTS struct{ ReadIntervalTimeout, ReadTotalTimeoutMultiplier, ReadTotalTimeoutConstant, WriteTotalTimeoutMultiplier, WriteTotalTimeoutConstant uint32 }

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	gdi32             = syscall.NewLazyDLL("gdi32.dll")
	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pCreateWindowExW  = user32.NewProc("CreateWindowExW")
	pDefWindowProcW   = user32.NewProc("DefWindowProcW")
	pShowWindow       = user32.NewProc("ShowWindow")
	pUpdateWindow     = user32.NewProc("UpdateWindow")
	pGetMessageW      = user32.NewProc("GetMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pSetWindowTextW   = user32.NewProc("SetWindowTextW")
	pGetWindowTextW   = user32.NewProc("GetWindowTextW")
	pSendMessageW     = user32.NewProc("SendMessageW")
	pLoadCursorW      = user32.NewProc("LoadCursorW")
	pLoadIconW        = user32.NewProc("LoadIconW")
	pLoadImageW       = user32.NewProc("LoadImageW")
	pSetTimer         = user32.NewProc("SetTimer")
	pKillTimer        = user32.NewProc("KillTimer")
	pMoveWindow       = user32.NewProc("MoveWindow")
	pGetClientRect    = user32.NewProc("GetClientRect")
	pMessageBoxW      = user32.NewProc("MessageBoxW")
	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGetCommState     = kernel32.NewProc("GetCommState")
	pSetCommState     = kernel32.NewProc("SetCommState")
	pSetCommTimeouts  = kernel32.NewProc("SetCommTimeouts")
	pPurgeComm        = kernel32.NewProc("PurgeComm")
	pRtlMoveMemory    = kernel32.NewProc("RtlMoveMemory")
	pCreateFontW      = gdi32.NewProc("CreateFontW")
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pSetBkColor       = gdi32.NewProc("SetBkColor")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
)

func w(s string) *uint16       { p, _ := syscall.UTF16PtrFromString(s); return p }
func rgb(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }
func i32(v int32) uintptr      { return uintptr(uint32(v)) }

var hwndMain uintptr
var ctr = map[string]uintptr{}
var hFont, hTitle, hValue uintptr
var brushWindow, brushWhite uintptr
var colorInk = rgb(39, 52, 67)
var colorMuted = rgb(88, 104, 123)
var colorAccent = rgb(47, 109, 179)
var colorTitle = rgb(22, 58, 99)

type Config struct {
	EagleURL, VirtualCOM, UpstreamCOM string
	Baud                              int
}

func defaults() Config { return Config{"http://127.0.0.1:1380", "COM20", "", 115200} }

var cfgMu sync.RWMutex
var cfg = defaults()

var mirror DewMirror
var runMu sync.Mutex
var running bool
var stopBridge chan struct{}
var virtualFile, upstreamFile *os.File
var statMu sync.RWMutex
var cpwiSeen bool
var lastCPWI time.Time
var rxCount, txCount, ignoredWrites, unknownCount uint64
var lastErr string
var logMu sync.Mutex
var logs []string

func appDataDir() string {
	if d := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); d != "" {
		return filepath.Join(d, "ECCO2CPWIDewMirror")
	}
	if d, err := os.UserConfigDir(); err == nil && strings.TrimSpace(d) != "" {
		return filepath.Join(d, "ECCO2CPWIDewMirror")
	}
	return filepath.Join(os.TempDir(), "ECCO2CPWIDewMirror")
}

func configPath() string     { return filepath.Join(appDataDir(), "ECCO2CPWIDewMirror.json") }
func startupLogPath() string { return filepath.Join(appDataDir(), "Logs", "startup.log") }

func legacyConfigPath() string {
	e, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(e), "ECCO2CPWIDewMirror.json")
}

func appIconPath() string {
	candidates := []string{}
	if e, err := os.Executable(); err == nil {
		dir := filepath.Dir(e)
		candidates = append(candidates,
			filepath.Join(dir, "ECCO2CPWIDewMirror.ico"),
			filepath.Join(dir, "assets", "ECCO2CPWIDewMirror.ico"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "ECCO2CPWIDewMirror.ico"),
			filepath.Join(wd, "assets", "ECCO2CPWIDewMirror.ico"),
		)
	}
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

func loadAppIcon() uintptr {
	if path := appIconPath(); path != "" {
		if h, _, _ := pLoadImageW.Call(0, uintptr(unsafe.Pointer(w(path))), IMAGE_ICON, 0, 0, LR_LOADFROMFILE|LR_DEFAULTSIZE); h != 0 {
			return h
		}
	}
	h, _, _ := pLoadIconW.Call(0, IDI_APPLICATION)
	return h
}

func writeStartupLog(message string) {
	path := startupLogPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	line := time.Now().Format(time.RFC3339Nano) + "  " + message + "\r\n"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func showStartupError(stage string, err error) {
	detail := stage
	if err != nil {
		detail += ": " + err.Error()
	}
	detail += "\r\n\r\nProtokoll: " + startupLogPath()
	writeStartupLog("ERROR " + detail)
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(w(detail))), uintptr(unsafe.Pointer(w(appTitle))), MB_OK|MB_ICONERROR|MB_SETFOREGROUND)
}

func loadConfig() {
	path := configPath()
	b, err := os.ReadFile(path)
	if err != nil {
		legacy := legacyConfigPath()
		if legacy != "" && legacy != path {
			if old, oldErr := os.ReadFile(legacy); oldErr == nil {
				b = old
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, old, 0o644)
			} else {
				return
			}
		} else {
			return
		}
	}
	var c Config
	if json.Unmarshal(b, &c) == nil {
		d := defaults()
		if c.EagleURL == "" {
			c.EagleURL = d.EagleURL
		}
		if c.VirtualCOM == "" {
			c.VirtualCOM = d.VirtualCOM
		}
		if c.Baud == 0 {
			c.Baud = d.Baud
		}
		cfgMu.Lock()
		cfg = c
		cfgMu.Unlock()
	}
}

func saveConfig() {
	readUiConfig()
	cfgMu.RLock()
	c := cfg
	cfgMu.RUnlock()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		appendLog("Konfiguration konnte nicht serialisiert werden: " + err.Error())
		return
	}
	path := configPath()
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		appendLog("Konfigurationsordner konnte nicht angelegt werden: " + err.Error())
		return
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0o644); err != nil {
		appendLog("Konfiguration konnte nicht geschrieben werden: " + err.Error())
		return
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		appendLog("Konfiguration konnte nicht ersetzt werden: " + err.Error())
		return
	}
	appendLog("Konfiguration gespeichert: " + path)
}

func appendLog(s string) {
	logMu.Lock()
	defer logMu.Unlock()
	line := time.Now().Format("15:04:05.000") + "  " + s
	logs = append(logs, line)
	if len(logs) > 250 {
		logs = logs[len(logs)-250:]
	}
}
func logText() string { logMu.Lock(); defer logMu.Unlock(); return strings.Join(logs, "\r\n") }

func readUi(id string) string {
	h := ctr[id]
	buf := make([]uint16, 512)
	r, _, _ := pGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:r])
}
func readUiConfig() {
	c := defaults()
	c.EagleURL = strings.TrimSpace(readUi("eagle"))
	c.VirtualCOM = strings.TrimSpace(readUi("vcom"))
	c.UpstreamCOM = strings.TrimSpace(readUi("upcom"))
	if v, e := strconv.Atoi(readUi("baud")); e == nil {
		c.Baud = v
	}
	cfgMu.Lock()
	cfg = c
	cfgMu.Unlock()
}

func getJSON(base, path string) (map[string]any, error) {
	cl := &http.Client{Timeout: 1200 * time.Millisecond}
	r, e := cl.Get(strings.TrimRight(base, "/") + path)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 65536))
	if e != nil {
		return nil, e
	}
	var m map[string]any
	e = json.Unmarshal(b, &m)
	return m, e
}
func num(m map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch x := v.(type) {
			case float64:
				return x, true
			case string:
				f, e := strconv.ParseFloat(strings.ReplaceAll(x, ",", "."), 64)
				if e == nil {
					return f, true
				}
			}
		}
	}
	return 0, false
}
func text(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func pollEagle() {
	cfgMu.RLock()
	base := cfg.EagleURL
	cfgMu.RUnlock()
	t := MirrorTelemetry{Updated: time.Now()}
	m, e := getJSON(base, "/getsupply")
	if e != nil {
		statMu.Lock()
		lastErr = "EAGLE: " + e.Error()
		statMu.Unlock()
		mirror.SetTelemetry(t)
		return
	}
	t.Online = true
	t.SupplyV, t.HasSupply = num(m, "supply")
	if m, e = getJSON(base, "/getecco"); e == nil {
		ecco := strings.ToLower(text(m, "ecco"))
		t.EccoOnline = strings.Contains(ecco, "connected") && !strings.Contains(ecco, "not")
		t.AmbientC, t.HasAmbient = num(m, "temp", "temperature")
		t.HumidityPct, t.HasHumidity = num(m, "hum", "humidity")
		t.DewPointC, t.HasDew = num(m, "dew", "dewpoint")
		t.T5C, t.HasT5 = num(m, "temp5", "t5")
		t.T6C, t.HasT6 = num(m, "temp6", "t6")
	}
	if m, e = getJSON(base, "/getregout?idx=1"); e == nil {
		t.Reg1V, t.HasReg1 = num(m, "voltage", "volt")
	}
	if m, e = getJSON(base, "/getregout?idx=2"); e == nil {
		t.Reg2V, t.HasReg2 = num(m, "voltage", "volt")
	}
	mirror.SetTelemetry(t)
	statMu.Lock()
	lastErr = ""
	statMu.Unlock()
}
func pollLoop() {
	for {
		pollEagle()
		time.Sleep(time.Second)
	}
}

func comPath(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if strings.HasPrefix(s, "\\\\.\\") {
		return s
	}
	return "\\\\.\\" + s
}
func openSerial(name string, baud int) (*os.File, error) {
	f, e := os.OpenFile(comPath(name), os.O_RDWR, 0)
	if e != nil {
		return nil, e
	}
	h := f.Fd()
	var d DCB
	d.DCBlength = uint32(unsafe.Sizeof(d))
	r, _, er := pGetCommState.Call(h, uintptr(unsafe.Pointer(&d)))
	if r == 0 {
		f.Close()
		return nil, er
	}
	d.BaudRate = uint32(baud)
	d.Flags |= 1
	d.ByteSize = 8
	d.Parity = 0
	d.StopBits = 0
	r, _, er = pSetCommState.Call(h, uintptr(unsafe.Pointer(&d)))
	if r == 0 {
		f.Close()
		return nil, er
	}
	to := COMMTIMEOUTS{ReadIntervalTimeout: 30, ReadTotalTimeoutConstant: 100, WriteTotalTimeoutConstant: 500}
	pSetCommTimeouts.Call(h, uintptr(unsafe.Pointer(&to)))
	pPurgeComm.Call(h, 0x0001|0x0002|0x0004|0x0008)
	return f, nil
}

func startBridge() {
	runMu.Lock()
	defer runMu.Unlock()
	if running {
		stopBridgeLocked()
		return
	}
	readUiConfig()
	cfgMu.RLock()
	c := cfg
	cfgMu.RUnlock()
	vf, e := openSerial(c.VirtualCOM, c.Baud)
	if e != nil {
		appendLog("Fehler beim Öffnen " + c.VirtualCOM + ": " + e.Error())
		statMu.Lock()
		lastErr = e.Error()
		statMu.Unlock()
		return
	}
	var uf *os.File
	if c.UpstreamCOM != "" {
		uf, e = openSerial(c.UpstreamCOM, c.Baud)
		if e != nil {
			vf.Close()
			appendLog("Upstream-Port konnte nicht geöffnet werden: " + e.Error())
			statMu.Lock()
			lastErr = e.Error()
			statMu.Unlock()
			return
		}
	}
	virtualFile = vf
	upstreamFile = uf
	stopBridge = make(chan struct{})
	running = true
	statMu.Lock()
	cpwiSeen = false
	rxCount = 0
	txCount = 0
	ignoredWrites = 0
	unknownCount = 0
	statMu.Unlock()
	appendLog("Mirror gestartet auf " + c.VirtualCOM + " (READ ONLY)")
	if c.UpstreamCOM != "" {
		appendLog("AUX-Passthrough zu " + c.UpstreamCOM + " aktiv")
	}
	go virtualReadLoop(vf, uf, stopBridge)
	if uf != nil {
		go upstreamReadLoop(vf, uf, stopBridge)
	}
}
func stopBridgeLocked() {
	if !running {
		return
	}
	close(stopBridge)
	if virtualFile != nil {
		virtualFile.Close()
	}
	if upstreamFile != nil {
		upstreamFile.Close()
	}
	virtualFile = nil
	upstreamFile = nil
	running = false
	appendLog("Mirror gestoppt")
}
func stopBridgeNow() { runMu.Lock(); defer runMu.Unlock(); stopBridgeLocked() }

func isRunning() bool {
	runMu.Lock()
	defer runMu.Unlock()
	return running
}

func virtualReadLoop(vf, uf *os.File, stop <-chan struct{}) {
	var dec PacketDecoder
	buf := make([]byte, 1024)
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, e := vf.Read(buf)
		if e != nil {
			if isRunning() {
				appendLog("COM Lesefehler: " + e.Error())
			}
			return
		}
		if n <= 0 {
			continue
		}
		packets := dec.Feed(buf[:n])
		for _, p := range packets {
			statMu.Lock()
			cpwiSeen = true
			lastCPWI = time.Now()
			rxCount++
			statMu.Unlock()
			reply, handled, desc := mirror.HandlePacket(p)
			if handled {
				appendLog("CPWI > " + packetHex(p) + "  " + desc)
				if strings.Contains(desc, "ignored") {
					statMu.Lock()
					ignoredWrites++
					statMu.Unlock()
				}
				if strings.HasPrefix(desc, "unknown") {
					statMu.Lock()
					unknownCount++
					statMu.Unlock()
				}
				if len(reply) > 0 {
					if _, e = vf.Write(reply); e == nil {
						statMu.Lock()
						txCount++
						statMu.Unlock()
						appendLog("CPWI < " + packetHex(reply))
					}
				}
			} else if uf != nil {
				_, _ = uf.Write(p)
				appendLog("AUX passthrough > " + packetHex(p))
			} else {
				appendLog("Nicht-Dew AUX-Paket ignoriert: " + packetHex(p))
			}
		}
	}
}
func upstreamReadLoop(vf, uf *os.File, stop <-chan struct{}) {
	buf := make([]byte, 2048)
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, e := uf.Read(buf)
		if e != nil {
			return
		}
		if n > 0 {
			_, _ = vf.Write(buf[:n])
		}
	}
}

func fmtTemp(v float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.2f °C", v)
}
func fmtPct(v float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.0f %%", v)
}
func pwmPct(t MirrorTelemetry, ch int) string {
	var v float64
	var ok bool
	if ch == 0 {
		v, ok = t.Reg1V, t.HasReg1
	} else {
		v, ok = t.Reg2V, t.HasReg2
	}
	if !ok || !t.HasSupply || t.SupplyV <= 0 {
		return "—"
	}
	p := v / t.SupplyV * 100
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return fmt.Sprintf("%.0f %%", p)
}

func setText(h uintptr, s string) { pSetWindowTextW.Call(h, uintptr(unsafe.Pointer(w(s)))) }
func updateUi() {
	t := mirror.Telemetry()
	setText(ctr["eagleStatus"], map[bool]string{true: "● EAGLE erreichbar", false: "● EAGLE nicht erreichbar"}[t.Online])
	setText(ctr["eccoStatus"], map[bool]string{true: "● ECCO2 verbunden", false: "● ECCO2 nicht verbunden"}[t.EccoOnline])
	setText(ctr["amb"], fmtTemp(t.AmbientC, t.HasAmbient))
	setText(ctr["hum"], fmtPct(t.HumidityPct, t.HasHumidity))
	setText(ctr["dew"], fmtTemp(t.DewPointC, t.HasDew))
	setText(ctr["t5"], fmtTemp(t.T5C, t.HasT5))
	setText(ctr["t6"], fmtTemp(t.T6C, t.HasT6))
	setText(ctr["h1"], pwmPct(t, 0))
	setText(ctr["h2"], pwmPct(t, 1))
	runMu.Lock()
	r := running
	runMu.Unlock()
	if r {
		setText(ctr["start"], "Mirror stoppen")
	} else {
		setText(ctr["start"], "Mirror starten")
	}
	statMu.RLock()
	seen := cpwiSeen
	age := time.Since(lastCPWI)
	rx, tx, iw, un, le := rxCount, txCount, ignoredWrites, unknownCount, lastErr
	statMu.RUnlock()
	cpwi := "● CPWI wartet"
	if seen && age < 5*time.Second {
		cpwi = "● CPWI aktiv"
	}
	setText(ctr["cpwiStatus"], cpwi)
	setText(ctr["stats"], fmt.Sprintf("RX %d   TX %d   ignorierte Schreibbefehle %d   unbekannt %d", rx, tx, iw, un))
	if le != "" {
		setText(ctr["error"], le)
	} else {
		setText(ctr["error"], "READ ONLY · keine /setregout-Aufrufe · ECCO2 bleibt alleiniger Regler")
	}
	setText(ctr["log"], logText())
}

func add(cls, text string, style uint32, id int) uintptr {
	h, _, err := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(w(cls))), uintptr(unsafe.Pointer(w(text))), uintptr(WS_CHILD|WS_VISIBLE|style), 0, 0, 0, 0, hwndMain, uintptr(id), 0, 0)
	if h == 0 {
		panic(fmt.Sprintf("Steuerelement %s konnte nicht erstellt werden: %v", cls, err))
	}
	pSendMessageW.Call(h, WM_SETFONT, hFont, 1)
	return h
}
func move(h uintptr, x, y, wid, hei int) {
	pMoveWindow.Call(h, uintptr(x), uintptr(y), uintptr(wid), uintptr(hei), 1)
}
func layout(cw, ch int) {
	m := 22
	if cw < 960 {
		cw = 960
	}
	w0 := cw - 2*m
	titleH := 38
	subH := 24
	move(ctr["title"], m, 14, w0, titleH)
	move(ctr["sub"], m, 54, w0, subH)
	y := 88
	buttonW := 132
	fieldH := 28
	buttonH := 32
	cfgH := 126
	move(ctr["cfgGroup"], m, y, w0, cfgH)
	move(ctr["eagleLab"], m+18, y+30, 92, fieldH)
	move(ctr["eagle"], m+112, y+30, w0-128-buttonW-14, fieldH)
	move(ctr["save"], m+w0-buttonW-18, y+28, buttonW, buttonH)
	move(ctr["vcomLab"], m+18, y+72, 92, fieldH)
	move(ctr["vcom"], m+112, y+72, 92, fieldH)
	move(ctr["upLab"], m+218, y+72, 150, fieldH)
	move(ctr["upcom"], m+372, y+72, 96, fieldH)
	move(ctr["baudLab"], m+484, y+72, 46, fieldH)
	move(ctr["baud"], m+534, y+72, 88, fieldH)
	move(ctr["start"], m+w0-buttonW-18, y+70, buttonW, buttonH)
	y += cfgH + 14
	half := (w0 - 14) / 2
	groupH := 236
	move(ctr["eccoGroup"], m, y, half, groupH)
	move(ctr["mirrorGroup"], m+half+14, y, half, groupH)
	lx := m + 18
	rowTop := y + 34
	rowStep := 27
	labelW := 142
	valueW := half - 176
	if valueW < 120 {
		valueW = 120
	}
	labels := []string{"ambLab", "humLab", "dewLab", "t5Lab", "t6Lab", "h1Lab", "h2Lab"}
	vals := []string{"amb", "hum", "dew", "t5", "t6", "h1", "h2"}
	for i := range labels {
		rowY := rowTop + i*rowStep
		move(ctr[labels[i]], lx, rowY, labelW, 24)
		move(ctr[vals[i]], lx+labelW+8, rowY, valueW, 24)
	}
	rx := m + half + 32
	statusW := half - 36
	move(ctr["eagleStatus"], rx, y+36, statusW, 28)
	move(ctr["eccoStatus"], rx, y+70, statusW, 28)
	move(ctr["cpwiStatus"], rx, y+104, statusW, 28)
	move(ctr["mode"], rx, y+142, statusW, 28)
	move(ctr["stats"], rx, y+178, statusW, 42)
	y += groupH + 14
	move(ctr["error"], m, y, w0, 30)
	y += 38
	logH := ch - y - m
	if logH < 170 {
		logH = 170
	}
	move(ctr["logGroup"], m, y, w0, logH)
	move(ctr["log"], m+14, y+28, w0-28, logH-44)
}

func buildUi() {
	ctr["title"] = add("STATIC", "ECCO2 CPWI Dew Mirror", SS_LEFT, 0)
	pSendMessageW.Call(ctr["title"], WM_SETFONT, hTitle, 1)
	ctr["sub"] = add("STATIC", "Read-only Smart DewHeater Controller 2X mirror · ECCO2/EAGLE2 → CPWI", SS_LEFT, 0)
	ctr["cfgGroup"] = add("BUTTON", "Verbindung", BS_GROUPBOX, 0)
	ctr["eagleLab"] = add("STATIC", "EAGLE API", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["eagle"] = add("EDIT", "", WS_BORDER|ES_AUTOHSCROLL, ID_EAGLE)
	ctr["save"] = add("BUTTON", "Speichern", BS_PUSHBUTTON|WS_TABSTOP, ID_SAVE)
	ctr["vcomLab"] = add("STATIC", "Mirror COM", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["vcom"] = add("EDIT", "", WS_BORDER|ES_AUTOHSCROLL, ID_VCOM)
	ctr["upLab"] = add("STATIC", "AUX-Passthrough optional", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["upcom"] = add("EDIT", "", WS_BORDER|ES_AUTOHSCROLL, ID_UPCOM)
	ctr["baudLab"] = add("STATIC", "Baud", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["baud"] = add("EDIT", "", WS_BORDER|ES_AUTOHSCROLL, 0)
	ctr["start"] = add("BUTTON", "Mirror starten", BS_PUSHBUTTON|WS_TABSTOP, ID_START)
	ctr["eccoGroup"] = add("BUTTON", "ECCO2 Telemetrie", BS_GROUPBOX, 0)
	names := []struct{ k, t string }{{"ambLab", "Umgebung"}, {"humLab", "Feuchte"}, {"dewLab", "Taupunkt"}, {"t5Lab", "T5 / Heater 1"}, {"t6Lab", "T6 / Heater 2"}, {"h1Lab", "Heater 1 Leistung"}, {"h2Lab", "Heater 2 Leistung"}}
	for _, n := range names {
		ctr[n.k] = add("STATIC", n.t, SS_LEFT|SS_CENTERIMAGE, 0)
	}
	for _, k := range []string{"amb", "hum", "dew", "t5", "t6", "h1", "h2"} {
		ctr[k] = add("STATIC", "—", SS_LEFT|SS_CENTERIMAGE, 0)
		pSendMessageW.Call(ctr[k], WM_SETFONT, hValue, 1)
	}
	ctr["mirrorGroup"] = add("BUTTON", "CPWI Mirror Status", BS_GROUPBOX, 0)
	ctr["eagleStatus"] = add("STATIC", "● EAGLE nicht erreichbar", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["eccoStatus"] = add("STATIC", "● ECCO2 nicht verbunden", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["cpwiStatus"] = add("STATIC", "● CPWI wartet", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["mode"] = add("STATIC", "🔒 Mirror Mode: READ ONLY", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["stats"] = add("STATIC", "RX 0   TX 0", SS_LEFT, 0)
	ctr["error"] = add("STATIC", "READ ONLY · ECCO2 bleibt alleiniger Regler", SS_LEFT|SS_CENTERIMAGE, 0)
	ctr["logGroup"] = add("BUTTON", "Protokoll", BS_GROUPBOX, 0)
	ctr["log"] = add("EDIT", "", WS_BORDER|ES_READONLY|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL, 0)
	cfgMu.RLock()
	c := cfg
	cfgMu.RUnlock()
	setText(ctr["eagle"], c.EagleURL)
	setText(ctr["vcom"], c.VirtualCOM)
	setText(ctr["upcom"], c.UpstreamCOM)
	setText(ctr["baud"], strconv.Itoa(c.Baud))
	var rc RECT
	pGetClientRect.Call(hwndMain, uintptr(unsafe.Pointer(&rc)))
	layout(int(rc.Right), int(rc.Bottom))
}

func setMinimumTrackSize(lParam uintptr, width, height int32) {
	// lParam is a Win32-owned pointer. Converting it directly from uintptr to
	// unsafe.Pointer is valid only for the duration of the callback, but Go's
	// unsafeptr analyser intentionally rejects that fragile pattern. Copy the
	// structure through RtlMoveMemory instead, update it locally, and copy it
	// back while the callback is active.
	var info MINMAXINFO
	size := unsafe.Sizeof(info)
	pRtlMoveMemory.Call(uintptr(unsafe.Pointer(&info)), lParam, size)
	info.PtMinTrackSize.X = width
	info.PtMinTrackSize.Y = height
	pRtlMoveMemory.Call(lParam, uintptr(unsafe.Pointer(&info)), size)
	runtime.KeepAlive(&info)
}

func wndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_GETMINMAXINFO:
		setMinimumTrackSize(lp, 960, 780)
		return 0
	case WM_SIZE:
		if wp != SIZE_MINIMIZED {
			var rc RECT
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			layout(int(rc.Right), int(rc.Bottom))
		}
		return 0
	case WM_COMMAND:
		id := int(wp & 0xffff)
		if id == ID_START {
			startBridge()
		} else if id == ID_SAVE {
			saveConfig()
		}
		return 0
	case WM_TIMER:
		updateUi()
		return 0
	case WM_CTLCOLORSTATIC:
		dc := wp
		ctl := lp
		if ctl == ctr["title"] {
			pSetTextColor.Call(dc, colorTitle)
		} else if ctl == ctr["sub"] {
			pSetTextColor.Call(dc, colorMuted)
		} else if ctl == ctr["error"] {
			pSetTextColor.Call(dc, rgb(166, 92, 0))
		} else if ctl == ctr["eagleStatus"] || ctl == ctr["eccoStatus"] || ctl == ctr["cpwiStatus"] {
			pSetTextColor.Call(dc, colorAccent)
		} else {
			pSetTextColor.Call(dc, colorInk)
		}
		pSetBkMode.Call(dc, TRANSPARENT)
		return brushWindow
	case WM_CTLCOLOREDIT:
		dc := wp
		pSetTextColor.Call(dc, colorInk)
		pSetBkColor.Call(dc, rgb(255, 255, 255))
		return brushWhite
	case WM_CLOSE:
		stopBridgeNow()
	case WM_DESTROY:
		pKillTimer.Call(hwnd, 1)
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func main() {
	// Win32 windows and their message queue are owned by the creating OS thread.
	// Lock the goroutine before the first USER32 call so CreateWindowExW and
	// GetMessageW always run on the same thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		if r := recover(); r != nil {
			showStartupError("Unerwarteter Startfehler", fmt.Errorf("%v", r))
		}
	}()

	writeStartupLog("START " + appTitle)
	loadConfig()

	hinst, _, hinstErr := pGetModuleHandleW.Call(0)
	if hinst == 0 {
		showStartupError("GetModuleHandleW fehlgeschlagen", hinstErr)
		return
	}

	hFont, _, _ = pCreateFontW.Call(i32(-16), 0, 0, 0, FW_NORMAL, 0, 0, 0, DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH|FF_DONTCARE, uintptr(unsafe.Pointer(w("Segoe UI"))))
	hTitle, _, _ = pCreateFontW.Call(i32(-25), 0, 0, 0, FW_SEMIBOLD, 0, 0, 0, DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH|FF_DONTCARE, uintptr(unsafe.Pointer(w("Segoe UI"))))
	hValue, _, _ = pCreateFontW.Call(i32(-17), 0, 0, 0, FW_SEMIBOLD, 0, 0, 0, DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH|FF_DONTCARE, uintptr(unsafe.Pointer(w("Segoe UI"))))
	brushWindow, _, _ = pCreateSolidBrush.Call(rgb(245, 247, 250))
	brushWhite, _, _ = pCreateSolidBrush.Call(rgb(255, 255, 255))

	cb := syscall.NewCallback(wndProc)
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	ico := loadAppIcon()
	classPtr := w(className)
	titlePtr := w(appTitle)
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), Style: 3, LpfnWndProc: cb, HInstance: hinst, HIcon: ico, HCursor: cur, HbrBackground: brushWindow, LpszClassName: classPtr, HIconSm: ico}
	atom, _, registerErr := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		showStartupError("Fensterklasse konnte nicht registriert werden", registerErr)
		return
	}

	createdWindow, _, createErr := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(classPtr)), uintptr(unsafe.Pointer(titlePtr)), WS_OVERLAPPEDWINDOW, uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT), 980, 820, 0, 0, hinst, 0)
	hwndMain = createdWindow
	if hwndMain == 0 {
		showStartupError("Statusfenster konnte nicht erstellt werden", createErr)
		return
	}

	buildUi()
	pSendMessageW.Call(hwndMain, WM_SETICON, ICON_SMALL, ico)
	pSendMessageW.Call(hwndMain, WM_SETICON, ICON_BIG, ico)
	appendLog("Gestartet. Mirror ist sicherheitsbedingt READ ONLY.")
	timer, _, timerErr := pSetTimer.Call(hwndMain, 1, 250, 0)
	if timer == 0 {
		showStartupError("Statusaktualisierung konnte nicht gestartet werden", timerErr)
		return
	}

	pShowWindow.Call(hwndMain, SW_SHOWNORMAL)
	pUpdateWindow.Call(hwndMain)
	writeStartupLog("WINDOW_VISIBLE")

	// Network polling starts only after the window is visible. A slow or faulty
	// EAGLE endpoint can therefore never suppress the status window.
	go pollLoop()

	var m MSG
	for {
		r, _, messageErr := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) == -1 {
			showStartupError("Windows-Nachrichtenschleife fehlgeschlagen", messageErr)
			break
		}
		if r == 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	stopBridgeNow()
	for _, h := range []uintptr{hFont, hTitle, hValue, brushWindow, brushWhite} {
		if h != 0 {
			pDeleteObject.Call(h)
		}
	}
	runtime.KeepAlive(cb)
	writeStartupLog("EXIT")
}
