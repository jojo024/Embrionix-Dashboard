package api

import (
	"net/http"

	"github.com/embrionix/dashboard/internal/api/handlers"
	"github.com/embrionix/dashboard/internal/api/middleware"
	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	cfg *config.Config,
	deviceSvc *services.DeviceService,
	pollingSvc *services.PollingService,
	pollRepo *repositories.PollRepository,
	authSvc *services.AuthService,
	userRepo *repositories.UserRepository,
	reportSvc *services.ReportService,
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
	authHandler := handlers.NewAuthHandler(authSvc, userRepo)
	reportHandler := handlers.NewReportHandler(reportSvc)

	v1 := r.Group("/api/v1")

	// Public: login (no auth required).
	v1.POST("/auth/login", authHandler.Login)

	// read = authenticated, any role (viewer+). When auth is disabled the
	// Authenticate middleware grants an implicit admin so nothing changes.
	read := v1.Group("", middleware.Authenticate(authSvc))
	// write = operator or admin. admin = admin only (user management).
	write := v1.Group("", middleware.Authenticate(authSvc), middleware.RequireRole(models.RoleOperator))
	admin := v1.Group("", middleware.Authenticate(authSvc), middleware.RequireRole(models.RoleAdmin))

	// --- Reads (viewer+) ---
	read.GET("/auth/me", authHandler.Me)
	read.GET("/devices", deviceHandler.ListDevices)
	read.GET("/devices/:id", deviceHandler.GetDevice)
	read.GET("/devices/:id/history", monHandler.GetDeviceHistory)
	read.GET("/devices/:id/history.csv", monHandler.ExportDeviceHistoryCSV)
	read.GET("/devices/:id/reachability", monHandler.GetDeviceReachability)
	read.GET("/devices/:id/config", monHandler.GetDeviceConfig)
	read.GET("/devices/:id/config/export", backupHandler.ExportDeviceConfig)
	read.GET("/audit", configWriteHandler.GetAuditLog)
	read.GET("/summary", deviceHandler.GetDeviceSummary)
	read.GET("/alarms", deviceHandler.GetFleetAlarms)
	read.GET("/alerts", monHandler.GetAlertHistory)
	read.GET("/settings/:key", settingsHandler.GetSetting)
	read.GET("/config", configHandler.GetConfig)
	read.GET("/export/ansible", deviceHandler.GetAnsibleInventory)
	read.GET("/report.pdf", reportHandler.GetReportPDF)

	// --- Writes & device actions (operator+) ---
	write.POST("/devices", deviceHandler.CreateDevice)
	write.PUT("/devices/:id", deviceHandler.UpdateDevice)
	write.DELETE("/devices/:id", deviceHandler.DeleteDevice)
	write.POST("/devices/:id/poll", monHandler.PollDeviceNow)
	write.PUT("/devices/:id/config/network", configWriteHandler.UpdateNetwork)
	write.PUT("/devices/:id/config/protocols", configWriteHandler.UpdateProtocols)
	write.PUT("/devices/:id/config/syslog", configWriteHandler.UpdateSyslog)
	write.PUT("/devices/:id/config/routes", configWriteHandler.UpdateRoutes)
	write.POST("/devices/:id/reboot", configWriteHandler.Reboot)
	write.POST("/devices/:id/config-reset", configWriteHandler.ConfigReset)
	write.POST("/devices/:id/config/import", backupHandler.ImportDeviceConfig)
	write.POST("/bulk/config", backupHandler.BulkConfig)
	write.GET("/backup", backupHandler.BackupDatabase) // full DB export → operator+
	write.PUT("/settings/:key", settingsHandler.SetSetting)

	// --- User management (admin only) ---
	admin.GET("/users", authHandler.ListUsers)
	admin.POST("/users", authHandler.CreateUser)
	admin.PUT("/users/:id", authHandler.UpdateUser)
	admin.DELETE("/users/:id", authHandler.DeleteUser)

	return r
}
