package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	deviceSvc  *services.DeviceService
	pollingSvc *services.PollingService
}

func NewDeviceHandler(deviceSvc *services.DeviceService, pollingSvc *services.PollingService) *DeviceHandler {
	return &DeviceHandler{deviceSvc: deviceSvc, pollingSvc: pollingSvc}
}

// ListDevices GET /api/v1/devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	devices, err := h.deviceSvc.ListDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich each device with live polling data
	for i := range devices {
		h.pollingSvc.EnrichDevice(&devices[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"total":   len(devices),
	})
}

// GetDevice GET /api/v1/devices/:id
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	device, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.pollingSvc.EnrichDevice(device)
	c.JSON(http.StatusOK, device)
}

// CreateDevice POST /api/v1/devices
// Requires name and at least one management IP (red or blue). Automatically fetches
// the firmware version from the device via the API.
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if device.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if device.ManagementIPRed == "" && device.ManagementIPBlue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of management_ip_red or management_ip_blue is required"})
		return
	}

	// Auto-fetch serial number and firmware version from the device
	ip := device.ManagementIPRed
	if ip == "" {
		ip = device.ManagementIPBlue
	}

	client := services.NewEmsfpClient(ip, "80", 10)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	info, err := client.FetchInfo(ctx)
	if err == nil {
		// Auto-populate firmware version from device; don't override if user provided it
		if device.FirmwareVersion == "" && info.CurrentVersion != "" {
			device.FirmwareVersion = info.CurrentVersion
		}
	}
	// If fetch fails, continue anyway (device may be offline at registration time)

	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()

	if err := h.deviceSvc.CreateDevice(&device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, device)
}

// UpdateDevice PUT /api/v1/devices/:id
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	existing, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.UpdatedAt = time.Now()

	if err := h.deviceSvc.UpdateDevice(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// DeleteDevice DELETE /api/v1/devices/:id
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	if err := h.deviceSvc.DeleteDevice(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetDeviceSummary GET /api/v1/summary
func (h *DeviceHandler) GetDeviceSummary(c *gin.Context) {
	counts := h.pollingSvc.Summary()
	devices, _ := h.deviceSvc.ListDevices()
	counts["total"] = len(devices)
	c.JSON(http.StatusOK, counts)
}

// ReportHandler serves the on-demand fleet report.
type ReportHandler struct {
	reportSvc *services.ReportService
}

func NewReportHandler(reportSvc *services.ReportService) *ReportHandler {
	return &ReportHandler{reportSvc: reportSvc}
}

// GetReportPDF GET /api/v1/report.pdf
func (h *ReportHandler) GetReportPDF(c *gin.Context) {
	pdf, err := h.reportSvc.PDF()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=embrionix-fleet-report.pdf")
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// GetAnsibleInventory GET /api/v1/export/ansible
// Returns the device inventory in Ansible dynamic-inventory JSON format so it
// can be consumed directly by `ansible-inventory` / playbooks.
func (h *DeviceHandler) GetAnsibleInventory(c *gin.Context) {
	devices, err := h.deviceSvc.ListDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hostvars := gin.H{}
	hosts := make([]string, 0, len(devices))
	for _, d := range devices {
		host := d.ManagementIPRed
		if host == "" {
			host = d.ManagementIPBlue
		}
		hostvars[d.Name] = gin.H{
			"ansible_host":        host,
			"management_ip_red":   d.ManagementIPRed,
			"management_ip_blue":  d.ManagementIPBlue,
			"embrionix_model":     d.Model,
			"embrionix_serial":    d.SerialNumber,
			"embrionix_location":  d.Location,
			"embrionix_rack":      d.Rack,
			"embrionix_tags":      d.Tags,
			"monitoring_enabled":  d.MonitoringEnabled,
		}
		hosts = append(hosts, d.Name)
	}

	c.Header("Content-Disposition", "attachment; filename=embrionix-inventory.json")
	c.JSON(http.StatusOK, gin.H{
		"_meta": gin.H{"hostvars": hostvars},
		"all":   gin.H{"hosts": hosts},
		"emsfp": gin.H{"hosts": hosts},
	})
}

// GetFleetAlarms GET /api/v1/alarms
// Returns every active alarm across the fleet for the dashboard alarm panel.
func (h *DeviceHandler) GetFleetAlarms(c *gin.Context) {
	devices, err := h.deviceSvc.ListDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	names := make(map[string]string, len(devices))
	for _, d := range devices {
		names[d.ID] = d.Name
	}
	alarms := h.pollingSvc.FleetAlarms(names)
	c.JSON(http.StatusOK, gin.H{"alarms": alarms, "total": len(alarms)})
}
