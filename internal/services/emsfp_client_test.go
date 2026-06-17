package services

import (
	"testing"

	"github.com/embrionix/dashboard/internal/models"
)

func TestBuildAlarms(t *testing.T) {
	data := &models.DevicePollingData{
		PTP:                 &models.PTPStatus{Locked: false, StatusLabel: "unlocked"},
		Ethernet:            &models.EthernetStats{RxError: "12"},
		VideoBandwidthUsage: "over",
		PortDetails: []models.PortDetail{
			{PortID: "p0", DDM: &models.SFPDDM{AlarmStatus: models.DDMAlarmStatus{LowRxPower: true}}},
		},
	}
	buildAlarms(data)
	if len(data.Alarms) != 4 {
		t.Fatalf("expected 4 alarms (ptp, eth, bw, port), got %d: %v", len(data.Alarms), data.Alarms)
	}

	clean := &models.DevicePollingData{PTP: &models.PTPStatus{Locked: true}, VideoBandwidthUsage: "good"}
	buildAlarms(clean)
	if len(clean.Alarms) != 0 {
		t.Fatalf("expected no alarms on a healthy device, got %v", clean.Alarms)
	}
}

func TestCarrySlowDataAndRebuildAlarms(t *testing.T) {
	// A previous full poll captured SFP DDM with a low-RX alarm + firmware.
	prev := &models.DevicePollingData{
		MACAddress:    "40:a3:6b:00:00:01",
		FirmwareSlots: []models.FirmwareSlot{{Slot: 1, Version: "2.0.1", Active: true}},
		PortDetails:   []models.PortDetail{{PortID: "p0", DDM: &models.SFPDDM{AlarmStatus: models.DDMAlarmStatus{LowRxPower: true}}}},
	}
	// A light poll only fetched fast data (e.g. PTP locked, good bandwidth).
	light := &models.DevicePollingData{PTP: &models.PTPStatus{Locked: true}, VideoBandwidthUsage: "good"}

	carrySlowData(light, prev)
	buildAlarms(light)

	if light.MACAddress != prev.MACAddress || len(light.FirmwareSlots) != 1 {
		t.Fatal("static fields were not carried forward")
	}
	// The carried port DDM alarm must still surface on the light poll.
	if len(light.Alarms) != 1 {
		t.Fatalf("expected the carried port alarm to be rebuilt, got %v", light.Alarms)
	}
}

func TestParseLLDPNeighbors(t *testing.T) {
	// Array form (multi-port device) — preserves per-interface mapping.
	arr := []byte(`[
		{"interface":3,"chassis":"28:99:3a:e7:56:e2","port":"Ethernet35","ttl":"120"},
		{"interface":5,"chassis":"28:99:3a:e7:56:e2","port":"Ethernet40","ttl":"120"}
	]`)
	got := parseLLDPNeighbors(arr)
	if len(got) != 2 {
		t.Fatalf("array form: expected 2 neighbors, got %d", len(got))
	}
	if got[0].Interface != 3 || got[0].PortID != "Ethernet35" || got[1].Interface != 5 {
		t.Errorf("array form: per-interface fields wrong: %+v", got)
	}

	// Single-object form (one-interface device).
	single := []byte(`{"chassis":"10:62:eb:d1:e9:00","port":"eth1/0/26","ttl":"120"}`)
	got = parseLLDPNeighbors(single)
	if len(got) != 1 || got[0].ChassisID != "10:62:eb:d1:e9:00" || got[0].PortID != "eth1/0/26" {
		t.Errorf("single form parsed wrong: %+v", got)
	}

	// Empty / missing.
	if n := parseLLDPNeighbors(nil); n != nil {
		t.Errorf("nil input should yield no neighbors, got %+v", n)
	}
}

func TestPingSucceeded(t *testing.T) {
	winReply := "Reply from 192.168.1.50: bytes=32 time=1ms TTL=64"
	winUnreachable := "Reply from 192.168.1.1: Destination host unreachable."
	winTimeout := "Request timed out."
	linuxReply := "64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.045 ms"

	if !pingSucceeded(winReply, true) {
		t.Error("windows echo reply should succeed")
	}
	if pingSucceeded(winUnreachable, true) {
		t.Error("'destination host unreachable' (no TTL) must not count as reachable")
	}
	if pingSucceeded(winTimeout, false) {
		t.Error("timeout (non-zero exit) must not count as reachable")
	}
	if !pingSucceeded(linuxReply, true) {
		t.Error("linux echo reply should succeed")
	}
}

func TestParsePingMs(t *testing.T) {
	cases := map[string]int64{
		"bytes=32 time=1ms TTL=64":        1,
		"bytes=32 time<1ms TTL=64":        1, // sub-ms clamps to 1
		"icmp_seq=1 ttl=64 time=0.045 ms": 1,
		"icmp_seq=1 ttl=64 time=12.6 ms":  13,
		"no timing here":                  0,
	}
	for out, want := range cases {
		if got := parsePingMs(out); got != want {
			t.Errorf("parsePingMs(%q) = %d, want %d", out, got, want)
		}
	}
}

func TestAsString(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{"80", "80"},          // string passes through
		{float64(80), "80"},   // JSON number -> string
		{float64(0), "0"},     // numeric VLAN id
		{nil, ""},             // missing key
		{true, ""},            // unsupported type -> empty
	}
	for _, c := range cases {
		if got := asString(c.in); got != c.want {
			t.Errorf("asString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPTPStatusLabel(t *testing.T) {
	cases := []struct {
		code      string
		wantLabel string
		wantLock  bool
	}{
		{"0", "unlocked", false},
		{"1", "coarse lock", false},
		{"3", "locked", true},
		{"0x3", "locked", true},
		{"", "unlocked", false},
		{"7", "code 7", false},
	}
	for _, c := range cases {
		label, locked := ptpStatusLabel(c.code)
		if label != c.wantLabel || locked != c.wantLock {
			t.Errorf("ptpStatusLabel(%q) = (%q, %v), want (%q, %v)",
				c.code, label, locked, c.wantLabel, c.wantLock)
		}
	}
}
