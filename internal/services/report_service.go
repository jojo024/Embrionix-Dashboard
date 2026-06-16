package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/pkg/logger"
	"github.com/go-pdf/fpdf"
)

// ReportService builds fleet-summary reports (on-demand PDF and a text summary
// delivered to the alerting webhook on a schedule).
type ReportService struct {
	deviceRepo *repositories.DeviceRepository
	pollRepo   *repositories.PollRepository
	pollingSvc *PollingService
	notifier   *Notifier
}

func NewReportService(deviceRepo *repositories.DeviceRepository, pollRepo *repositories.PollRepository, pollingSvc *PollingService, notifier *Notifier) *ReportService {
	return &ReportService{deviceRepo: deviceRepo, pollRepo: pollRepo, pollingSvc: pollingSvc, notifier: notifier}
}

// ReportData is the assembled content of a fleet report.
type ReportData struct {
	GeneratedAt time.Time
	Counts      map[string]int
	Total       int
	Alarms      []FleetAlarm
	Transitions []models.AlertEvent
}

func (s *ReportService) buildData() ReportData {
	devices, _ := s.deviceRepo.FindAll()
	names := make(map[string]string, len(devices))
	for _, d := range devices {
		names[d.ID] = d.Name
	}
	counts := s.pollingSvc.Summary()
	alarms := s.pollingSvc.FleetAlarms(names)
	transitions, _ := s.pollRepo.FindAlerts("", 15)

	return ReportData{
		GeneratedAt: time.Now(),
		Counts:      counts,
		Total:       len(devices),
		Alarms:      alarms,
		Transitions: transitions,
	}
}

// Text renders a compact plain-text summary (used for webhook delivery).
func (s *ReportService) Text() string { return renderText(s.buildData()) }

// PDF renders the fleet report as a PDF document.
func (s *ReportService) PDF() ([]byte, error) { return renderPDF(s.buildData()) }

func renderText(d ReportData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Embrionix fleet report — %s\n", d.GeneratedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "Devices: %d total — %d online, %d warning, %d critical, %d offline\n",
		d.Total, d.Counts["online"], d.Counts["warning"], d.Counts["critical"], d.Counts["offline"])
	if len(d.Alarms) == 0 {
		b.WriteString("No active alarms.\n")
	} else {
		fmt.Fprintf(&b, "Active alarms (%d):\n", len(d.Alarms))
		for i, a := range d.Alarms {
			if i >= 10 {
				fmt.Fprintf(&b, "  …and %d more\n", len(d.Alarms)-10)
				break
			}
			fmt.Fprintf(&b, "  - %s: %s\n", a.DeviceName, a.Message)
		}
	}
	return b.String()
}

func renderPDF(d ReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, "Embrionix Fleet Report", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(0, 6, "Generated "+d.GeneratedAt.Format("2006-01-02 15:04 MST"), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	// Summary table
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Fleet status", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	summaryRows := [][2]string{
		{"Total devices", itoa(d.Total)},
		{"Online", itoa(d.Counts["online"])},
		{"Warning", itoa(d.Counts["warning"])},
		{"Critical", itoa(d.Counts["critical"])},
		{"Offline", itoa(d.Counts["offline"])},
		{"Unknown", itoa(d.Counts["unknown"])},
	}
	for _, r := range summaryRows {
		pdf.CellFormat(50, 7, r[0], "B", 0, "L", false, 0, "")
		pdf.CellFormat(0, 7, r[1], "B", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	// Active alarms
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Active alarms (%d)", len(d.Alarms)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	if len(d.Alarms) == 0 {
		pdf.CellFormat(0, 6, "No active alarms.", "", 1, "L", false, 0, "")
	} else {
		for _, a := range d.Alarms {
			pdf.CellFormat(60, 6, truncate(a.DeviceName, 32), "", 0, "L", false, 0, "")
			pdf.CellFormat(0, 6, truncate(a.Message, 90), "", 1, "L", false, 0, "")
		}
	}
	pdf.Ln(4)

	// Recent status changes
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Recent status changes", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	if len(d.Transitions) == 0 {
		pdf.CellFormat(0, 6, "None recorded.", "", 1, "L", false, 0, "")
	} else {
		for _, t := range d.Transitions {
			pdf.CellFormat(40, 6, t.CreatedAt.Format("01-02 15:04"), "", 0, "L", false, 0, "")
			pdf.CellFormat(50, 6, truncate(t.DeviceName, 28), "", 0, "L", false, 0, "")
			pdf.CellFormat(0, 6, fmt.Sprintf("%s -> %s", t.FromStatus, t.ToStatus), "", 1, "L", false, 0, "")
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DeliverScheduled builds the text report and posts it to the webhook.
func (s *ReportService) DeliverScheduled() {
	if !s.notifier.Enabled() {
		logger.Warn("scheduled report skipped: no webhook configured")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s.notifier.NotifyText(ctx, s.Text())
	logger.Info("scheduled fleet report delivered")
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
