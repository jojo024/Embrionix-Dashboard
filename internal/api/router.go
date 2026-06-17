package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/embrionix/dashboard/internal/api/handlers"
	"github.com/embrionix/dashboard/internal/api/middleware"
	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/embrionix/dashboard/internal/webui"
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
	updateSvc *services.UpdateService,
) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// Health endpoint
	r.GET("/health", handlers.HealthCheck)

	// Serve the embedded frontend (single self-contained binary). Unmatched API
	// paths get a JSON 404; everything else falls back to the SPA's index.html.
	staticFS := webui.FS()
	hasUI := webui.Available()
	fileServer := http.FileServer(http.FS(staticFS))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/health" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if !hasUI {
			c.JSON(http.StatusNotFound, gin.H{"error": "frontend not built into this binary"})
			return
		}
		// Serve the requested asset if it exists; otherwise SPA-fallback to index.html.
		clean := strings.TrimPrefix(path.Clean(p), "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(staticFS, clean); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	deviceHandler := handlers.NewDeviceHandler(deviceSvc, pollingSvc)
	monHandler := handlers.NewMonitoringHandler(deviceSvc, pollingSvc, pollRepo, cfg.Polling.TimeoutSeconds)
	settingsHandler := handlers.NewSettingsHandler(pollRepo)
	configHandler := handlers.NewConfigHandler(cfg)
	configWriteHandler := handlers.NewConfigWriteHandler(deviceSvc, pollRepo, cfg.Polling.TimeoutSeconds)
	backupHandler := handlers.NewBackupHandler(deviceSvc, pollRepo, configWriteHandler, cfg.Polling.TimeoutSeconds)
	authHandler := handlers.NewAuthHandler(authSvc, userRepo)
	reportHandler := handlers.NewReportHandler(reportSvc)
	updateHandler := handlers.NewUpdateHandler(updateSvc)

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
	read.GET("/version", updateHandler.GetVersion)

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
	write.POST("/update/check", updateHandler.CheckUpdate) // force a release re-check

	// --- User management & self-update (admin only) ---
	admin.GET("/users", authHandler.ListUsers)
	admin.POST("/users", authHandler.CreateUser)
	admin.PUT("/users/:id", authHandler.UpdateUser)
	admin.DELETE("/users/:id", authHandler.DeleteUser)
	admin.POST("/update", updateHandler.ApplyUpdate) // self-update + restart

	return r
}
