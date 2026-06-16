package api

import (
	"net/http"

	"github.com/embrionix/dashboard/internal/api/handlers"
	"github.com/embrionix/dashboard/internal/api/middleware"
	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	cfg *config.Config,
	deviceSvc *services.DeviceService,
	pollingSvc *services.PollingService,
	pollRepo *repositories.PollRepository,
) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// Health endpoint
	r.GET("/health", handlers.HealthCheck)

	// Serve embedded frontend (in production, the web/dist is embedded)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	deviceHandler := handlers.NewDeviceHandler(deviceSvc, pollingSvc)
	monHandler := handlers.NewMonitoringHandler(deviceSvc, pollingSvc, pollRepo, cfg.Polling.TimeoutSeconds)
	settingsHandler := handlers.NewSettingsHandler(pollRepo)
	configHandler := handlers.NewConfigHandler(cfg)
	configWriteHandler := handlers.NewConfigWriteHandler(deviceSvc, pollRepo, cfg.Polling.TimeoutSeconds)
	backupHandler := handlers.NewBackupHandler(deviceSvc, pollRepo, configWriteHandler, cfg.Polling.TimeoutSeconds)

	v1 := r.Group("/api/v1")
	{
		// Devices CRUD
		v1.GET("/devices", deviceHandler.ListDevices)
		v1.POST("/devices", deviceHandler.CreateDevice)
		v1.GET("/devices/:id", deviceHandler.GetDevice)
		v1.PUT("/devices/:id", deviceHandler.UpdateDevice)
		v1.DELETE("/devices/:id", deviceHandler.DeleteDevice)

		// Monitoring
		v1.GET("/devices/:id/history", monHandler.GetDeviceHistory)
		v1.GET("/devices/:id/history.csv", monHandler.ExportDeviceHistoryCSV)
		v1.POST("/devices/:id/poll", monHandler.PollDeviceNow)
		v1.GET("/devices/:id/reachability", monHandler.GetDeviceReachability)
		v1.GET("/devices/:id/config", monHandler.GetDeviceConfig)

		// Configuration writes + device actions (Phase 4b) — audited
		v1.PUT("/devices/:id/config/network", configWriteHandler.UpdateNetwork)
		v1.PUT("/devices/:id/config/protocols", configWriteHandler.UpdateProtocols)
		v1.PUT("/devices/:id/config/syslog", configWriteHandler.UpdateSyslog)
		v1.PUT("/devices/:id/config/routes", configWriteHandler.UpdateRoutes)
		v1.POST("/devices/:id/reboot", configWriteHandler.Reboot)
		v1.POST("/devices/:id/config-reset", configWriteHandler.ConfigReset)
		v1.GET("/audit", configWriteHandler.GetAuditLog)

		// Backup / restore / bulk (Phase 4c)
		v1.GET("/devices/:id/config/export", backupHandler.ExportDeviceConfig)
		v1.POST("/devices/:id/config/import", backupHandler.ImportDeviceConfig)
		v1.GET("/backup", backupHandler.BackupDatabase)
		v1.POST("/bulk/config", backupHandler.BulkConfig)

		// Dashboard summary + fleet-wide alarms + alert history
		v1.GET("/summary", deviceHandler.GetDeviceSummary)
		v1.GET("/alarms", deviceHandler.GetFleetAlarms)
		v1.GET("/alerts", monHandler.GetAlertHistory)

		// Settings + effective runtime config
		v1.GET("/settings/:key", settingsHandler.GetSetting)
		v1.PUT("/settings/:key", settingsHandler.SetSetting)
		v1.GET("/config", configHandler.GetConfig)
	}

	return r
}
