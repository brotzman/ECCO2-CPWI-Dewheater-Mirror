package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	auxStart = 0x3b
	dewMain  = 0x17
	dewBoot  = 0xbb

	cmdGetVersion       = 0xfe
	cmdGetNumPorts      = 0x10
	cmdQueryPort        = 0x11
	cmdQueryHeater      = 0x12
	cmdEnablePort       = 0x14
	cmdSetAutoAgg       = 0x16
	cmdSetManualPWM     = 0x17
	cmdQueryEnvironment = 0x18
	cmdRecalibrate      = 0x19
	cmdQueryCalibrated  = 0x1a
	cmdInputPower       = 0x00
	cmdInputLimits      = 0x04
)

type MirrorTelemetry struct {
	Online      bool
	EccoOnline  bool
	AmbientC    float64
	HumidityPct float64
	DewPointC   float64
	T5C         float64
	T6C         float64
	HasAmbient  bool
	HasHumidity bool
	HasDew      bool
	HasT5       bool
	HasT6       bool
	SupplyV     float64
	HasSupply   bool
	Reg1V       float64
	Reg2V       float64
	HasReg1     bool
	HasReg2     bool
	Updated     time.Time
}

type DewMirror struct {
	mu sync.RWMutex
	t  MirrorTelemetry
}

func (d *DewMirror) SetTelemetry(t MirrorTelemetry) { d.mu.Lock(); d.t = t; d.mu.Unlock() }
func (d *DewMirror) Telemetry() MirrorTelemetry     { d.mu.RLock(); defer d.mu.RUnlock(); return d.t }

func checksum(pktWithoutStart []byte) byte {
	var sum byte
	for _, b := range pktWithoutStart {
		sum += b
	}
	return byte(0 - sum)
}

func buildPacket(src, dst, cmd byte, payload []byte) []byte {
	n := 3 + len(payload)
	p := make([]byte, 0, n+3)
	p = append(p, auxStart, byte(n), src, dst, cmd)
	p = append(p, payload...)
	p = append(p, checksum(p[1:]))
	return p
}

func validPacket(p []byte) bool {
	if len(p) < 6 || p[0] != auxStart {
		return false
	}
	want := int(p[1]) + 3 // start + len + body(len) + checksum
	if len(p) != want {
		return false
	}
	var sum byte
	for _, b := range p[1:] {
		sum += b
	}
	return sum == 0
}

func packetHex(p []byte) string {
	if len(p) == 0 {
		return ""
	}
	s := strings.ToUpper(hex.EncodeToString(p))
	var b strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[i : i+2])
	}
	return b.String()
}

func int32Millis(v float64) []byte {
	x := int64(math.Round(v * 1000.0))
	if x > math.MaxInt32 {
		x = math.MaxInt32
	}
	if x < math.MinInt32 {
		x = math.MinInt32
	}
	u := uint32(int32(x))
	return []byte{byte(u >> 24), byte(u >> 16), byte(u >> 8), byte(u)}
}

func u16(v int) []byte {
	if v < 0 {
		v = 0
	}
	if v > 65535 {
		v = 65535
	}
	return []byte{byte(v >> 8), byte(v)}
}

func clampByte(v float64) byte {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(math.Round(v))
}

func heaterPWM(t MirrorTelemetry, channel int) byte {
	var v float64
	var ok bool
	if channel == 0 {
		v, ok = t.Reg1V, t.HasReg1
	} else {
		v, ok = t.Reg2V, t.HasReg2
	}
	if !ok || !t.HasSupply || t.SupplyV <= 0.1 {
		return 0
	}
	return clampByte(v / t.SupplyV * 255.0)
}

func (d *DewMirror) HandlePacket(req []byte) (reply []byte, handled bool, desc string) {
	if !validPacket(req) {
		return nil, false, "ungültiges AUX-Paket"
	}
	src, dst, cmd := req[2], req[3], req[4]
	if dst != dewBoot && dst != dewMain {
		return nil, false, "anderes AUX-Gerät"
	}
	payload := req[5 : len(req)-1]
	t := d.Telemetry()

	var out []byte
	switch cmd {
	case cmdGetVersion:
		// Celestron 2X reference traces identify version 1.1.1270.
		out = []byte{0x01, 0x01, 0x04, 0xf6}
		desc = "GET_VERSION"
	case cmdGetNumPorts:
		out = []byte{0x03} // 2 dew + 1 accessory
		desc = "GET_NUM_PORTS"
	case cmdQueryPort:
		if len(payload) < 1 {
			out = []byte{0}
			desc = "QUERY_PORT missing"
			break
		}
		switch payload[0] {
		case 0x00:
			out = []byte{0x01}
			desc = "QUERY_PORT: accessory count"
		case 0x01:
			out = []byte{0x02}
			desc = "QUERY_PORT: dew count"
		case 0xff:
			out = []byte{0x00}
			desc = "QUERY_PORT: USB hub"
		default:
			// One fixed 12V accessory output. Celestron numbering starts at request 0x02;
			// the response identifies the port as 0x10 | (request-1), e.g. 0x11.
			pnum := payload[0] - 1
			out = []byte{0x10 | pnum, 0x01, 0x00, 0x00, 0x00}
			mv := 12000
			if t.HasSupply {
				mv = int(math.Round(t.SupplyV * 1000))
			}
			out = append(out, u16(mv)...)
			desc = fmt.Sprintf("QUERY_PORT: accessory %d", payload[0])
		}
	case cmdQueryEnvironment:
		amb := 0.0
		dew := 0.0
		hum := 0.0
		if t.HasAmbient {
			amb = t.AmbientC
		}
		if t.HasDew {
			dew = t.DewPointC
		}
		if t.HasHumidity {
			hum = t.HumidityPct
		}
		out = append(out, int32Millis(amb)...)
		out = append(out, int32Millis(dew)...)
		out = append(out, clampByte(hum))
		desc = "QUERY_ENVIRONMENT"
	case cmdQueryHeater:
		if len(payload) < 1 {
			out = []byte{0}
			desc = "QUERY_HEATER missing"
			break
		}
		ch := int(payload[0])
		if ch < 0 || ch > 1 {
			ch = 1
		}
		out = append(out, byte(ch+1))
		out = append(out, 0x01) // report auto mode; real control remains ECCO2
		out = append(out, heaterPWM(t, ch))
		out = append(out, 0x00, 0x00, 0x05) // reserved/reserved/aggression
		if ch == 0 && t.HasT5 {
			out = append(out, int32Millis(t.T5C)...)
		}
		if ch == 1 && t.HasT6 {
			out = append(out, int32Millis(t.T6C)...)
		}
		desc = fmt.Sprintf("QUERY_HEATER #%d", ch+1)
	case cmdInputPower:
		mv := 12000
		if t.HasSupply {
			mv = int(math.Round(t.SupplyV * 1000))
		}
		// Current is deliberately reported as 0: EAGLE2 API does not expose reliable total input current here.
		out = append(out, u16(mv)...)
		out = append(out, 0x00, 0x00, 0x00, 0x00)
		desc = "QUERY_INPUT_POWER"
	case cmdInputLimits:
		out = append(out, u16(4000)...)
		out = append(out, u16(10000)...)
		desc = "QUERY_INPUT_LIMITS"
	case cmdQueryCalibrated:
		out = []byte{0x01}
		desc = "QUERY_ENV_CALIBRATED"
	case cmdEnablePort, cmdSetAutoAgg, cmdSetManualPWM, cmdRecalibrate:
		// READ-ONLY MIRROR: acknowledge protocol traffic but never change ECCO/EAGLE hardware.
		out = nil
		desc = fmt.Sprintf("WRITE 0x%02X ignored (READ ONLY)", cmd)
	default:
		// Empty response keeps discovery from hanging while making unknown commands visible in the log.
		out = nil
		desc = fmt.Sprintf("unknown 0x%02X -> empty ACK", cmd)
	}
	return buildPacket(dst, src, cmd, out), true, desc
}

// AUX packet stream decoder. The AUX format is 3B LEN SRC DST CMD [DATA...] CHECKSUM,
// where LEN is SRC+DST+CMD+DATA bytes and the byte sum from LEN through CHECKSUM is 0 mod 256.
type PacketDecoder struct{ buf []byte }

func (d *PacketDecoder) Feed(in []byte) [][]byte {
	d.buf = append(d.buf, in...)
	var out [][]byte
	for {
		for len(d.buf) > 0 && d.buf[0] != auxStart {
			d.buf = d.buf[1:]
		}
		if len(d.buf) < 2 {
			break
		}
		total := int(d.buf[1]) + 3
		if total < 6 || total > 260 {
			d.buf = d.buf[1:]
			continue
		}
		if len(d.buf) < total {
			break
		}
		p := append([]byte(nil), d.buf[:total]...)
		d.buf = d.buf[total:]
		if validPacket(p) {
			out = append(out, p)
		}
	}
	return out
}
