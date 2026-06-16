package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

// ConfigWriteHandler performs configuration writes and device actions against
// the emSFP device, recording every attempt (success or failure) in the audit
// log. Writes happen only in response to an explicit request from the UI.
type ConfigWriteHandler struct {
	deviceSvc  *services.DeviceService
	pollRepo   *repositories.PollRepository
	timeoutSec int
}

func NewConfigWriteHandler(deviceSvc *services.DeviceService, pollRepo *repositories.PollRepository, timeoutSec int) *ConfigWriteHandler {
	return &ConfigWriteHandler{deviceSvc: deviceSvc, pollRepo: pollRepo, timeoutSec: timeoutSec}
}

// resolve looks up the device and its management IP, writing an error response
// and returning ok=false when either is missing.
func (h *ConfigWriteHandler) resolve(c *gin.Context) (device *models.Device, ip string, ok bool) {
	d, err := h.deviceSvc.GetDevice(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return nil, "", false
	}
	ip = d.ManagementIPRed
	if ip == "" {
		ip = d.ManagementIPBlue
	}
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no management IP configured"})
		return nil, "", false
	}
	return d, ip, true
}

// audit records the outcome of a write/action and shapes the HTTP response.
func (h *ConfigWriteHandler) audit(c *gin.Context, device *models.Device, action, detail string, err error) {
	event := &models.AuditEvent{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Action:     action,
		Detail:     detail,
		Success:    err == nil,
		CreatedAt:  time.Now(),
	}
	if err != nil {
		event.Message = err.Error()
	}
	_ = h.pollRepo.SaveAudit(event)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "audit": event})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "audit": event})
}

func (h *ConfigWriteHandler) clientFor(ip string) *services.EmsfpClient {
	return services.NewEmsfpClient(ip, "80", h.timeoutSec)
}

func (h *ConfigWriteHandler) ctx(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), time.Duration(h.timeoutSec)*time.Second)
}

// UpdateNetwork PUT /api/v1/devices/:id/config/network
func (h *ConfigWriteHandler) UpdateNetwork(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	var body models.NetworkUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Static config requires valid IPv4; DHCP mode does not.
	if body.DHCPEnable != "1" {
		for label, v := range map[string]string{"ip_addr": body.IPAddress, "subnet_mask": body.SubnetMask, "gateway": body.Gateway} {
			if !isIPv4(v) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s must be a valid IPv4 address", label)})
				return
			}
		}
	}
	if body.Port != "" {
		if p, err := strconv.Atoi(body.Port); err != nil || p < 1 || p > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "port must be 1-65535"})
			return
		}
	}

	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).UpdateNetwork(ctx, body)
	detail := fmt.Sprintf("network: dhcp=%s ip=%s gw=%s vlan=%s", dashIfEmpty(body.DHCPEnable), dashIfEmpty(body.IPAddress), dashIfEmpty(body.Gateway), dashIfEmpty(body.CtlVLANID))
	h.audit(c, device, "config.network", detail, err)
}

// UpdateProtocols PUT /api/v1/devices/:id/config/protocols
func (h *ConfigWriteHandler) UpdateProtocols(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	var body models.ProtocolsConfig
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).UpdateProtocols(ctx, body)
	detail := fmt.Sprintf("protocols: mdns=%s ember_port=%s sap=%s", body.MDNSEnable, body.EmberServerPort, body.SAPAnnouncementEnable)
	h.audit(c, device, "config.protocols", detail, err)
}

// UpdateSyslog PUT /api/v1/devices/:id/config/syslog
func (h *ConfigWriteHandler) UpdateSyslog(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	var body models.SyslogUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Enable && !isIPv4(body.Server) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "syslog server must be a valid IPv4 address when enabled"})
		return
	}
	if body.Port < 1 || body.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "syslog port must be 1-65535"})
		return
	}
	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).UpdateSyslog(ctx, body)
	detail := fmt.Sprintf("syslog: enable=%t server=%s:%d", body.Enable, dashIfEmpty(body.Server), body.Port)
	h.audit(c, device, "config.syslog", detail, err)
}

// UpdateRoutes PUT /api/v1/devices/:id/config/routes
func (h *ConfigWriteHandler) UpdateRoutes(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	var body struct {
		Routes []models.StaticRoute `json:"routes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(body.Routes) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at most 5 static routes are supported"})
		return
	}
	for _, r := range body.Routes {
		if _, _, err := net.ParseCIDR(r.Destination); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("destination %q must be CIDR (e.g. 192.168.1.0/24)", r.Destination)})
			return
		}
		if !isIPv4(r.Gateway) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("gateway %q must be a valid IPv4 address", r.Gateway)})
			return
		}
	}
	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).UpdateStaticRoutes(ctx, body.Routes)
	h.audit(c, device, "config.routes", fmt.Sprintf("static routes: %d configured", len(body.Routes)), err)
}

// Reboot POST /api/v1/devices/:id/reboot
func (h *ConfigWriteHandler) Reboot(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).Reboot(ctx)
	h.audit(c, device, "reboot", "device reboot requested", err)
}

// ConfigReset POST /api/v1/devices/:id/config-reset
func (h *ConfigWriteHandler) ConfigReset(c *gin.Context) {
	device, ip, ok := h.resolve(c)
	if !ok {
		return
	}
	var body struct {
		Scope string `json:"scope" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	valid := map[string]bool{"flows": true, "application": true, "generic": true, "system": true}
	if !valid[body.Scope] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be one of: flows, application, generic, system"})
		return
	}
	ctx, cancel := h.ctx(c)
	defer cancel()
	err := h.clientFor(ip).ConfigReset(ctx, body.Scope)
	h.audit(c, device, "config_reset", "config reset: "+body.Scope, err)
}

// GetAuditLog GET /api/v1/audit
func (h *ConfigWriteHandler) GetAuditLog(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	events, err := h.pollRepo.FindAudit(c.Query("device"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
}

func isIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
