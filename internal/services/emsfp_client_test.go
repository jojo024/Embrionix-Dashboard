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
