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
	const fast = int64(50)

	t.Run("clean device is online", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60}
		if got := s.deriveStatus(pd, fast); got != models.StatusOnline {
			t.Fatalf("got %q, want online", got)
		}
	})

	t.Run("alarms trigger warning", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60, Alarms: []string{"PTP not locked"}}
		if got := s.deriveStatus(pd, fast); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("warm device warns at the configured threshold", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 72}
		if got := s.deriveStatus(pd, fast); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("slow response warns", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 60}
		if got := s.deriveStatus(pd, 3000); got != models.StatusWarning {
			t.Fatalf("got %q, want warning", got)
		}
	})

	t.Run("hot device is critical and gets an alarm appended", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 80}
		got := s.deriveStatus(pd, fast)
		if got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
		}
		if len(pd.Alarms) == 0 {
			t.Fatalf("expected a temperature alarm to be appended")
		}
	})

	t.Run("critical outranks existing warnings", func(t *testing.T) {
		pd := &models.DevicePollingData{CoreTemp: 90, Alarms: []string{"some warning"}}
		if got := s.deriveStatus(pd, fast); got != models.StatusCritical {
			t.Fatalf("got %q, want critical", got)
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
