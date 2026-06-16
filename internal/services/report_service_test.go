package services

import (
	"strings"
	"testing"
	"time"

	"github.com/embrionix/dashboard/internal/models"
)

func sampleReport() ReportData {
	return ReportData{
		GeneratedAt: time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC),
		Total:       3,
		Counts:      map[string]int{"online": 1, "warning": 1, "critical": 1},
		Alarms: []FleetAlarm{
			{DeviceName: "Encap-1", Message: "PTP not locked"},
		},
		Transitions: []models.AlertEvent{
			{DeviceName: "Decap-2", FromStatus: models.StatusOnline, ToStatus: models.StatusCritical, CreatedAt: time.Now()},
		},
	}
}

func TestRenderText(t *testing.T) {
	out := renderText(sampleReport())
	if !strings.Contains(out, "3 total") {
		t.Errorf("text report missing device total: %q", out)
	}
	if !strings.Contains(out, "Encap-1") || !strings.Contains(out, "PTP not locked") {
		t.Errorf("text report missing the active alarm: %q", out)
	}
}

func TestRenderPDF(t *testing.T) {
	pdf, err := renderPDF(sampleReport())
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 {
		t.Fatalf("PDF looks empty (%d bytes)", len(pdf))
	}
	if string(pdf[:4]) != "%PDF" {
		t.Fatalf("output is not a PDF (header %q)", string(pdf[:4]))
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("short strings should pass through")
	}
	if r := truncate("hello world", 5); len([]rune(r)) != 5 {
		t.Errorf("expected 5 runes, got %q", r)
	}
}
