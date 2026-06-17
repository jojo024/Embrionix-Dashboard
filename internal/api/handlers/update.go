package handlers

import (
	"net/http"

	"github.com/embrionix/dashboard/internal/services"
	"github.com/embrionix/dashboard/internal/version"
	"github.com/gin-gonic/gin"
)

// UpdateHandler exposes version info and the self-update trigger.
type UpdateHandler struct {
	updateSvc *services.UpdateService
}

func NewUpdateHandler(updateSvc *services.UpdateService) *UpdateHandler {
	return &UpdateHandler{updateSvc: updateSvc}
}

// GetVersion GET /api/v1/version
// Returns the running version and the cached update-availability status.
func (h *UpdateHandler) GetVersion(c *gin.Context) {
	st := h.updateSvc.Status()
	st.CurrentVersion = version.Version
	c.JSON(http.StatusOK, st)
}

// CheckUpdate POST /api/v1/update/check
// Forces an immediate re-check against GitHub Releases (operator+).
func (h *UpdateHandler) CheckUpdate(c *gin.Context) {
	c.JSON(http.StatusOK, h.updateSvc.Check(c.Request.Context()))
}

// ApplyUpdate POST /api/v1/update
// Downloads and applies the latest release, then restarts the server (admin).
func (h *UpdateHandler) ApplyUpdate(c *gin.Context) {
	if err := h.updateSvc.Apply(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// The binary is swapped; the process will relaunch momentarily.
	c.JSON(http.StatusOK, gin.H{
		"status":  "updating",
		"message": "Update applied. The server is restarting; the page will reload shortly.",
	})
}
