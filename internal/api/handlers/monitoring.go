package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

type MonitoringHandler struct {
	deviceSvc  *services.DeviceService
	pollingSvc *services.PollingService
	pollRepo   *repositories.PollRepository
	pollCfg    struct{ TimeoutSeconds int }
}

func NewMonitoringHandler(
	deviceSvc *services.DeviceService,
	pollingSvc *services.PollingService,
	pollRepo *repositories.PollRepository,
	timeoutSec int,
) *MonitoringHandler {
	h := &MonitoringHandler{
		deviceSvc:  deviceSvc,
		pollingSvc: pollingSvc,
		pollRepo:   pollRepo,
	}
	h.pollCfg.TimeoutSeconds = timeoutSec
	return h
}

// GetDeviceHistory GET /api/v1/devices/:id/history
func (h *MonitoringHandler) GetDeviceHistory(c *gin.Context) {
	deviceID := c.Param("id")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	// If "since" query param given, use time range
	if sinceStr := c.Query("since"); sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since format, use RFC3339"})
			return
		}
		results, err := h.pollRepo.FindByDeviceSince(deviceID, since)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
		return
	}

	results, err := h.pollRepo.FindByDevice(deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

// PollDeviceNow POST /api/v1/devices/:id/poll
// Triggers an immediate poll of the specified device.
func (h *MonitoringHandler) PollDeviceNow(c *gin.Context) {
	device, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ip := device.ManagementIPRed
	if ip == "" {
		ip = device.ManagementIPBlue
	}
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no management IP configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.pollCfg.TimeoutSeconds)*time.Second)
	defer cancel()

	client := services.NewEmsfpClient(ip, "80", h.pollCfg.TimeoutSeconds)
	data, err := client.Poll(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"reachable": false,
			"error":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reachable":    true,
		"polling_data": data,
	})
}

// GetAlertHistory GET /api/v1/alerts
// Returns the status-transition alert history. Optional `device` query param
// scopes to one device; `limit` caps the count (default 100).
func (h *MonitoringHandler) GetAlertHistory(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	events, err := h.pollRepo.FindAlerts(c.Query("device"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": events, "total": len(events)})
}

// ExportDeviceHistoryCSV GET /api/v1/devices/:id/history.csv
// Streams poll history as a CSV download.
func (h *MonitoringHandler) ExportDeviceHistoryCSV(c *gin.Context) {
	deviceID := c.Param("id")
	limit := 1000
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	results, err := h.pollRepo.FindByDevice(deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=history-%s.csv", deviceID))

	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write([]string{
		"polled_at", "reachable", "response_ms", "core_temp", "fan_speed",
		"core_voltage", "port0_tx_power", "port0_rx_power", "ptp_locked", "ptp_offset",
	})
	for _, r := range results {
		_ = w.Write([]string{
			r.PolledAt.Format(time.RFC3339),
			strconv.FormatBool(r.Reachable),
			strconv.FormatInt(r.ResponseMs, 10),
			floatPtr(r.CoreTemp), intPtr(r.FanSpeed), intPtr(r.CoreVoltage),
			intPtr(r.Port0TxPower), intPtr(r.Port0RxPower),
			boolPtr(r.PTPLocked), int64Ptr(r.PTPOffset),
		})
	}
}

func floatPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}

func intPtr(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func int64Ptr(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}

func boolPtr(v *bool) string {
	if v == nil {
		return ""
	}
	return strconv.FormatBool(*v)
}

// GetDeviceConfig GET /api/v1/devices/:id/config
// Fetches the device's read-only configuration on demand (GET-only; never
// writes to the device).
func (h *MonitoringHandler) GetDeviceConfig(c *gin.Context) {
	device, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ip := device.ManagementIPRed
	if ip == "" {
		ip = device.ManagementIPBlue
	}
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no management IP configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.pollCfg.TimeoutSeconds)*time.Second)
	defer cancel()

	client := services.NewEmsfpClient(ip, "80", h.pollCfg.TimeoutSeconds)
	cfg, err := client.FetchConfig(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"reachable": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// GetDeviceReachability GET /api/v1/devices/:id/reachability
func (h *MonitoringHandler) GetDeviceReachability(c *gin.Context) {
	device, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	results := gin.H{}

	if device.ManagementIPRed != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.pollCfg.TimeoutSeconds)*time.Second)
		defer cancel()
		client := services.NewEmsfpClient(device.ManagementIPRed, "80", h.pollCfg.TimeoutSeconds)
		reachable, ms, _ := client.CheckReachability(ctx)
		results["red"] = gin.H{"ip": device.ManagementIPRed, "reachable": reachable, "response_ms": ms}
	}

	if device.ManagementIPBlue != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.pollCfg.TimeoutSeconds)*time.Second)
		defer cancel()
		client := services.NewEmsfpClient(device.ManagementIPBlue, "80", h.pollCfg.TimeoutSeconds)
		reachable, ms, _ := client.CheckReachability(ctx)
		results["blue"] = gin.H{"ip": device.ManagementIPBlue, "reachable": reachable, "response_ms": ms}
	}

	c.JSON(http.StatusOK, results)
}
