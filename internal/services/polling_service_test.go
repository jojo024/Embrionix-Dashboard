package services

import (
	"testing"

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
		if got := s.deriveStatus(pd, notSlow); got != models.StatusOnline {
			t.Fatalf("got %q, want online", got)
		}
	})

	t.Run("warm device warns at the configured threshold", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 72, PTP: lockedPTP}
		if got := s.deriveStatus(pd, notSlow); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("transient slowness does NOT warn (backoff)", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: lockedPTP}
		if got := s.deriveStatus(pd, SlowWarnAfter-1); got != models.StatusOnline {
			t.Fatalf("got %q, want online (under backoff threshold)", got)
		}
	})

	t.Run("sustained slowness warns", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: lockedPTP}
		if got := s.deriveStatus(pd, SlowWarnAfter); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("PTP not locked is critical", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, PTP: &models.PTPStatus{Locked: false}}
		if got := s.deriveStatus(pd, notSlow); got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
	})

	t.Run("populated SFP port with link down is critical", func(t *testing.T) {
		pd := &models.DevicePollingData{
			CoreTemp: 60, PTP: lockedPTP,
			PortDetails: []models.PortDetail{{PortID: "3", Link: "down", DDM: &models.SFPDDM{}}},
		}
		if got := s.deriveStatus(pd, notSlow); got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
	})

	t.Run("empty cage with no link does not flag", func(t *testing.T) {
		// No DDM = no module installed; link down is expected, not an alarm.
		pd := &models.DevicePollingData{
			CoreTemp: 60, PTP: lockedPTP,
			PortDetails: []models.PortDetail{{PortID: "1", Link: "down"}},
		}
		if got := s.deriveStatus(pd, notSlow); got != models.StatusOnline {
			t.Fatalf("got %q, want online", got)
		}
	})

	t.Run("hot device is critical and gets an alarm appended", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 80, PTP: lockedPTP}
		got := s.deriveStatus(pd, notSlow)
		if got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
		if len(pd.Alarms) == 0 {
			t.Fatalf("expected a temperature alarm to be appended")
		}
	})
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
