package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// HealthCheck GET /health
func HealthCheck(c *gin.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
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
