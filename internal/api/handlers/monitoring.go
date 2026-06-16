package handlers

import (
	"context"
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
