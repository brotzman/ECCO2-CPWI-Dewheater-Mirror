package main

import "testing"

func TestKnownPackets(t *testing.T) {
	p := []byte{0x3b, 0x03, 0x0d, 0xbb, 0xfe, 0x37}
	if !validPacket(p) {
		t.Fatal("known GET_VERSION packet invalid")
	}
	d := &DewMirror{}
	r, h, _ := d.HandlePacket(p)
	if !h || packetHex(r) != "3B 07 BB 0D FE 01 01 04 F6 37" {
		t.Fatalf("unexpected reply: %s", packetHex(r))
	}
}
func TestEnvironment(t *testing.T) {
	d := &DewMirror{}
	d.SetTelemetry(MirrorTelemetry{HasAmbient: true, AmbientC: 23.488, HasDew: true, DewPointC: 12.318, HasHumidity: true, HumidityPct: 49})
	req := buildPacket(0x0d, dewMain, cmdQueryEnvironment, nil)
	r, h, _ := d.HandlePacket(req)
	if !h {
		t.Fatal("not handled")
	}
	want := "3B 0C 17 0D 18 00 00 5B C0 00 00 30 1E 31 1E"
	if packetHex(r) != want {
		t.Fatalf("got %s want %s", packetHex(r), want)
	}
}
func TestDecoder(t *testing.T) {
	p := buildPacket(0x0d, dewMain, cmdGetNumPorts, nil)
	var d PacketDecoder
	a := d.Feed(append([]byte{0, 1, 2}, p[:3]...))
	if len(a) != 0 {
		t.Fatal("premature")
	}
	a = d.Feed(p[3:])
	if len(a) != 1 || packetHex(a[0]) != packetHex(p) {
		t.Fatalf("decode %v", a)
	}
}

func TestPublished2XTraces(t *testing.T) {
	d := &DewMirror{}
	d.SetTelemetry(MirrorTelemetry{HasAmbient: true, AmbientC: 23.488, HasDew: true, DewPointC: 12.318, HasHumidity: true, HumidityPct: 49, HasT5: true, T5C: 23.160, HasSupply: true, SupplyV: 11.87, HasReg1: true, Reg1V: 2.14})
	cases := []struct {
		req        []byte
		wantPrefix string
	}{
		{[]byte{0x3b, 0x03, 0x0d, 0x17, 0x10, 0xc9}, "3B 04 17 0D 10 03"},
		{[]byte{0x3b, 0x04, 0x0d, 0x17, 0x11, 0x00, 0xc7}, "3B 04 17 0D 11 01"},
		{[]byte{0x3b, 0x04, 0x0d, 0x17, 0x11, 0x01, 0xc6}, "3B 04 17 0D 11 02"},
		{[]byte{0x3b, 0x03, 0x0d, 0x17, 0x18, 0xc1}, "3B 0C 17 0D 18 00 00 5B C0 00 00 30 1E 31"},
		{[]byte{0x3b, 0x04, 0x0d, 0x17, 0x12, 0x00, 0xc6}, "3B 0D 17 0D 12 01 01"},
	}
	for _, c := range cases {
		r, h, _ := d.HandlePacket(c.req)
		if !h {
			t.Fatalf("not handled %s", packetHex(c.req))
		}
		got := packetHex(r)
		if len(got) < len(c.wantPrefix) || got[:len(c.wantPrefix)] != c.wantPrefix {
			t.Fatalf("req %s got %s want prefix %s", packetHex(c.req), got, c.wantPrefix)
		}
	}
}
