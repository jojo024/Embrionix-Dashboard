package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

// BackupHandler implements configuration backup/restore, database backup, and
// bulk configuration. Writes reuse ConfigWriteHandler's client + audit helpers.
type BackupHandler struct {
	deviceSvc  *services.DeviceService
	pollRepo   *repositories.PollRepository
	write      *ConfigWriteHandler
	timeoutSec int
}

func NewBackupHandler(deviceSvc *services.DeviceService, pollRepo *repositories.PollRepository, write *ConfigWriteHandler, timeoutSec int) *BackupHandler {
	return &BackupHandler{deviceSvc: deviceSvc, pollRepo: pollRepo, write: write, timeoutSec: timeoutSec}
}

// ExportDeviceConfig GET /api/v1/devices/:id/config/export
// Downloads the device's read-only configuration as a JSON snapshot.
func (h *BackupHandler) ExportDeviceConfig(c *gin.Context) {
	device, ip, ok := h.write.resolve(c)
	if !ok {
		return
	}
	ctx, cancel := h.write.ctx(c)
	defer cancel()
	cfg, err := h.write.clientFor(ip).FetchConfig(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	snapshot := gin.H{
		"version":     1,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"device_name": device.Name,
		"config":      cfg,
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=config-%s.json", device.ID))
	c.JSON(http.StatusOK, snapshot)
}

// ImportDeviceConfig POST /api/v1/devices/:id/config/import
// Applies a previously exported snapshot. Network is only applied when
// include_network=true (it reboots the device); protocols/syslog/routes always.
func (h *BackupHandler) ImportDeviceConfig(c *gin.Context) {
	device, ip, ok := h.write.resolve(c)
	if !ok {
		return
	}
	var body struct {
		IncludeNetwork bool                 `json:"include_network"`
		Config         models.DeviceConfig  `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := h.write.ctx(c)
	defer cancel()
	client := h.write.clientFor(ip)
	cfg := body.Config

	var applied []string
	var failed []string
	record := func(section string, err error) {
		ev := &models.AuditEvent{
			DeviceID: device.ID, DeviceName: device.Name,
			Action: "config.import." + section, Detail: "restored from snapshot",
			Success: err == nil, CreatedAt: time.Now(),
		}
		if err != nil {
			ev.Message = err.Error()
			failed = append(failed, section)
		} else {
			applied = append(applied, section)
		}
		_ = h.pollRepo.SaveAudit(ev)
	}

	if cfg.Protocols != nil {
		record("protocols", client.UpdateProtocols(ctx, *cfg.Protocols))
	}
	if cfg.Syslog != nil {
		record("syslog", client.UpdateSyslog(ctx, models.SyslogUpdate{
			Server: cfg.Syslog.Server, Port: cfg.Syslog.Port, Enable: cfg.Syslog.Enable, Monitoring: cfg.Syslog.Monitoring,
		}))
	}
	if cfg.StaticRoutes != nil {
		record("routes", client.UpdateStaticRoutes(ctx, cfg.StaticRoutes))
	}
	if body.IncludeNetwork && cfg.Network != nil {
		record("network", client.UpdateNetwork(ctx, models.NetworkUpdate{
			IPAddress: cfg.Network.IPAddress, SubnetMask: cfg.Network.SubnetMask, Gateway: cfg.Network.Gateway,
			Hostname: cfg.Network.Hostname, Port: cfg.Network.Port, DHCPEnable: cfg.Network.DHCPEnable,
			CtlVLANID: cfg.Network.CtlVLANID, CtlVLANPCP: cfg.Network.CtlVLANPCP, CtlVLANEnable: cfg.Network.CtlVLANEnable,
		}))
	}

	c.JSON(http.StatusOK, gin.H{"applied": applied, "failed": failed})
}

// BackupDatabase GET /api/v1/backup
// Streams a consistent snapshot of the SQLite database (VACUUM INTO).
func (h *BackupHandler) BackupDatabase(c *gin.Context) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("embrionix-backup-%d.db", time.Now().UnixNano()))
	if err := h.pollRepo.BackupTo(tmp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.Remove(tmp)

	filename := fmt.Sprintf("embrionix-%s.db", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.File(tmp)
}

// BulkConfig POST /api/v1/bulk/config
// Applies one configuration section to multiple devices. Each device is written
// concurrently and audited individually.
func (h *BackupHandler) BulkConfig(c *gin.Context) {
	var body struct {
		DeviceIDs []string             `json:"device_ids" binding:"required"`
		Section   string               `json:"section" binding:"required"` // protocols | syslog
		Protocols *models.ProtocolsConfig `json:"protocols"`
		Syslog    *models.SyslogUpdate    `json:"syslog"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Section != "protocols" && body.Section != "syslog" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "section must be 'protocols' or 'syslog'"})
		return
	}

	type result struct {
		DeviceID string `json:"device_id"`
		Success  bool   `json:"success"`
		Message  string `json:"message,omitempty"`
	}
	results := make([]result, len(body.DeviceIDs))

	var wg sync.WaitGroup
	for i, id := range body.DeviceIDs {
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := result{DeviceID: id}
			device, err := h.deviceSvc.GetDevice(id)
			if err != nil {
				res.Message = "device not found"
				results[i] = res
				return
			}
			ip := device.ManagementIPRed
			if ip == "" {
				ip = device.ManagementIPBlue
			}
			if ip == "" {
				res.Message = "no management IP"
				results[i] = res
				return
			}

			ctx, cancel := h.write.ctx(c)
			defer cancel()
			client := h.write.clientFor(ip)

			switch body.Section {
			case "protocols":
				if body.Protocols == nil {
					err = fmt.Errorf("protocols payload missing")
				} else {
					err = client.UpdateProtocols(ctx, *body.Protocols)
				}
			case "syslog":
				if body.Syslog == nil {
					err = fmt.Errorf("syslog payload missing")
				} else {
					err = client.UpdateSyslog(ctx, *body.Syslog)
				}
			}

			ev := &models.AuditEvent{
				DeviceID: device.ID, DeviceName: device.Name,
				Action: "bulk." + body.Section, Detail: "bulk apply",
				Success: err == nil, CreatedAt: time.Now(),
			}
			if err != nil {
				ev.Message = err.Error()
				res.Message = err.Error()
			} else {
				res.Success = true
			}
			_ = h.pollRepo.SaveAudit(ev)
			results[i] = res
		}()
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"results": results})
}
