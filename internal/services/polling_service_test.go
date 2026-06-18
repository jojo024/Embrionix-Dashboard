package services

import (
	"testing"
	"time"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
)

func testService() *PollingService {
	return &PollingService{
		alertCfg: config.AlertingConfig{
			TempWarningC:   70,
			TempCriticalC:  75,
			ResponseWarnMs: 2000,
		},
	}
}

func TestDeriveStatus(t *testing.T) {
	s := testService()
	const notSlow = 0
	lockedPTP := &models.PTPStatus{Locked: true}

	t.Run("clean device is online", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: lockedPTP}
		if got := s.deriveStatus(pd, notSlow, nil); got != models.StatusOnline {
			t.Fatalf("got %q, want online", got)
		}
	})

	t.Run("warm device warns at the configured threshold", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 72, PTP: lockedPTP}
		if got := s.deriveStatus(pd, notSlow, nil); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("transient slowness does NOT warn (backoff)", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: lockedPTP}
		if got := s.deriveStatus(pd, SlowWarnAfter-1, nil); got != models.StatusOnline {
			t.Fatalf("got %q, want online (under backoff threshold)", got)
		}
	})

	t.Run("sustained slowness warns", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: lockedPTP}
		if got := s.deriveStatus(pd, SlowWarnAfter, nil); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("PTP not locked is critical", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: &models.PTPStatus{Locked: false}}
		if got := s.deriveStatus(pd, notSlow, nil); got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
	})

	t.Run("populated SFP port with link down is critical", func(t *testing.T) {
		pd := &models.DevicePollingData{
			CoreTemp: 60, PTP: lockedPTP,
			PortDetails: []models.PortDetail{{PortID: "3", Link: "down", DDM: &models.SFPDDM{}}},
		}
		if got := s.deriveStatus(pd, notSlow, nil); got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
	})

	t.Run("empty cage with no link does not flag", func(t *testing.T) {
		// No DDM = no module installed; link down is expected, not an alarm.
		pd := &models.DevicePollingData{
			CoreTemp: 60, PTP: lockedPTP,
			PortDetails: []models.PortDetail{{PortID: "1", Link: "down"}},
		}
		if got := s.deriveStatus(pd, notSlow, nil); got != models.StatusOnline {
			t.Fatalf("got %q, want online", got)
		}
	})

	t.Run("hot device is critical and gets an alarm appended", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 80, PTP: lockedPTP}
		got := s.deriveStatus(pd, notSlow, nil)
		if got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
		if len(pd.Alarms) == 0 {
			t.Fatalf("expected a temperature alarm to be appended")
		}
	})
}

func TestMicroWattToDBm(t *testing.T) {
	if _, ok := microWattToDBm(0); ok {
		t.Error("0 µW should be invalid")
	}
	dbm, ok := microWattToDBm(353)
	if !ok || dbm < -4.6 || dbm > -4.4 { // 353 µW ≈ -4.52 dBm
		t.Errorf("353 µW: got %.2f dBm (ok=%v), want ≈ -4.52", dbm, ok)
	}
}

func TestDeriveStatusTxPower(t *testing.T) {
	s := &PollingService{alertCfg: config.AlertingConfig{
		TempWarningC: 70, TempCriticalC: 75,
		TxPowerWarnDBm: -6, TxPowerCritDBm: -9, TxPowerPorts: []int{3, 5},
	}}
	locked := &models.PTPStatus{Locked: true}

	cases := []struct {
		name   string
		ports  []models.PortTelemetry
		expect models.DeviceStatus
	}{
		{"healthy TX (~-3 dBm)", []models.PortTelemetry{{Port: 3, TxPower: 500}}, models.StatusOnline},
		{"low TX (~-7 dBm) warns", []models.PortTelemetry{{Port: 3, TxPower: 200}}, models.StatusWarning},
		{"very low TX (~-10 dBm) critical", []models.PortTelemetry{{Port: 5, TxPower: 100}}, models.StatusCritical},
		{"low TX on unmonitored port ignored", []models.PortTelemetry{{Port: 1, TxPower: 100}}, models.StatusOnline},
		{"no module (0 µW) ignored", []models.PortTelemetry{{Port: 3, TxPower: 0}}, models.StatusOnline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pd := &models.DevicePollingData{CoreTemp: 60, PTP: locked, Ports: c.ports}
			if got := s.deriveStatus(pd, 0, nil); got != c.expect {
				t.Fatalf("got %q, want %q", got, c.expect)
			}
		})
	}
}

func TestReservePoll(t *testing.T) {
	s := &PollingService{lastPoll: make(map[string]time.Time)}
	if ok, _ := s.reservePoll("d1"); !ok {
		t.Fatal("first poll should be allowed")
	}
	if ok, wait := s.reservePoll("d1"); ok || wait <= 0 {
		t.Fatalf("second immediate poll should be blocked with a positive wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := s.reservePoll("d2"); !ok {
		t.Fatal("a different device should be allowed independently")
	}
}

func TestStaggerStep(t *testing.T) {
	if staggerStep(1, 30) != 0 || staggerStep(0, 30) != 0 {
		t.Error("0 or 1 device should have no stagger")
	}
	// 1000 devices over 15s would be 15ms each — under the 250ms cap.
	if got := staggerStep(1000, 30); got <= 0 || got > 250*time.Millisecond {
		t.Errorf("got %v, want a small positive step ≤ 250ms", got)
	}
	// Few devices: 4 over 15s = ~3.75s each, capped at 250ms.
	if got := staggerStep(4, 30); got != 250*time.Millisecond {
		t.Errorf("got %v, want 250ms (capped)", got)
	}
}

func TestFleetAlarms(t *testing.T) {
	s := &PollingService{results: map[string]*pollState{
		"d1": {
			Reachable: true,
			Status:    models.StatusWarning,
			Data:      &models.DevicePollingData{Alarms: []string{"PTP not locked", "RX error"}},
		},
		"d2": {Reachable: false, Status: models.StatusOffline},
		"d3": {Reachable: true, Status: models.StatusOnline, Data: &models.DevicePollingData{}},
	}}

	alarms := s.FleetAlarms(map[string]string{"d1": "Encap-1", "d2": "Decap-2"})

	// d1 contributes 2 alarms, d2 contributes 1 (unreachable), d3 contributes 0.
	if len(alarms) != 3 {
		t.Fatalf("expected 3 fleet alarms, got %d", len(alarms))
	}

	var sawUnreachable bool
	for _, a := range alarms {
		if a.DeviceID == "d2" {
			sawUnreachable = true
			if a.DeviceName != "Decap-2" {
				t.Errorf("expected device name resolved to Decap-2, got %q", a.DeviceName)
			}
			if a.Status != models.StatusOffline {
				t.Errorf("expected offline status for unreachable device, got %q", a.Status)
			}
		}
	}
	if !sawUnreachable {
		t.Error("expected an unreachable alarm for d2")
	}
}
