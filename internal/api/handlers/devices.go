package handlers

import (
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
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
