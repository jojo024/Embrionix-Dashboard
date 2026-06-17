package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/version"
	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// ConfigHandler exposes the effective, non-sensitive runtime configuration so
// the UI can display the active polling and alerting settings (configured via
// config.yaml / env). The webhook URL is reported only as enabled/disabled.
type ConfigHandler struct{ cfg *config.Config }

func NewConfigHandler(cfg *config.Config) *ConfigHandler { return &ConfigHandler{cfg: cfg} }

// GetConfig GET /api/v1/config
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	p := h.cfg.Polling
	a := h.cfg.Alerting
	c.JSON(http.StatusOK, gin.H{
		"polling": gin.H{
			"interval_seconds":       p.IntervalSeconds,
			"timeout_seconds":        p.TimeoutSeconds,
			"icmp_enabled":           p.ICMPEnabled,
			"history_retention_days": p.HistoryRetentionDays,
		},
		"alerting": gin.H{
			"temp_warning_c":      a.TempWarningC,
			"temp_critical_c":     a.TempCriticalC,
			"response_warning_ms": a.ResponseWarnMs,
			"tx_power_warn_dbm":   a.TxPowerWarnDBm,
			"tx_power_crit_dbm":   a.TxPowerCritDBm,
			"tx_power_ports":      a.TxPowerPorts,
			"webhook_enabled":     a.WebhookURL != "",
			"webhook_on":          a.WebhookOn,
		},
	})
}

// HealthCheck GET /health
func HealthCheck(c *gin.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"version":   version.Version,
		"uptime":    time.Since(startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"go_version": runtime.Version(),
		"memory_mb": mem.Alloc / 1024 / 1024,
	})
}

// SettingsHandler GET/PUT /api/v1/settings/:key
type SettingsHandler struct {
	pollRepo interface {
		GetSetting(key string) (string, error)
		SetSetting(key, value string) error
	}
}

func NewSettingsHandler(pollRepo interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}) *SettingsHandler {
	return &SettingsHandler{pollRepo: pollRepo}
}

func (h *SettingsHandler) GetSetting(c *gin.Context) {
	val, err := h.pollRepo.GetSetting(c.Param("key"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": c.Param("key"), "value": val})
}

func (h *SettingsHandler) SetSetting(c *gin.Context) {
	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.pollRepo.SetSetting(c.Param("key"), body.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": c.Param("key"), "value": body.Value})
}
