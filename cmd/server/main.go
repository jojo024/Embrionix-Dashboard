package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/embrionix/dashboard/internal/api"
	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/embrionix/dashboard/pkg/database"
	"github.com/embrionix/dashboard/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// seedAdmin creates the initial admin account the first time auth is enabled
// (when no users exist). The password comes from config; if unset, a random one
// is generated and logged once so the operator can capture it.
func seedAdmin(userRepo *repositories.UserRepository, authCfg config.AuthConfig) error {
	n, err := userRepo.Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	password := authCfg.AdminPassword
	generated := false
	if password == "" {
		buf := make([]byte, 12)
		if _, err := rand.Read(buf); err != nil {
			return err
		}
		password = hex.EncodeToString(buf)
		generated = true
	}

	hash, err := services.HashPassword(password)
	if err != nil {
		return err
	}
	user := &models.User{
		Username:     authCfg.AdminUsername,
		PasswordHash: hash,
		Role:         models.RoleAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := userRepo.Create(user); err != nil {
		return err
	}

	if generated {
		logger.Warn("seeded initial admin user with a GENERATED password — change it after first login",
			zap.String("username", authCfg.AdminUsername),
			zap.String("password", password),
		)
	} else {
		logger.Info("seeded initial admin user from config", zap.String("username", authCfg.AdminUsername))
	}
	return nil
}

// listenWithRetry binds addr, retrying for up to timeout. This lets a freshly
// relaunched instance (after a self-update) wait for the old process to release
// the port instead of failing immediately.
func listenWithRetry(addr string, timeout time.Duration) (net.Listener, error) {
	deadline := time.Now().Add(timeout)
	for {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	cfgPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	// Ensure working directories exist (out-of-box ready).
	for _, dir := range []string{"data", "logs"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "config loaded from %s\n", cfgPath)

	if err := logger.Init(cfg.Logging.Level, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&models.Device{}, &models.PollResult{}, &models.AppSetting{}, &models.AlertEvent{}, &models.AuditEvent{}, &models.User{}); err != nil {
		logger.Fatal("failed to migrate database", zap.Error(err))
	}

	deviceRepo := repositories.NewDeviceRepository(db)
	pollRepo := repositories.NewPollRepository(db)
	userRepo := repositories.NewUserRepository(db)
	deviceSvc := services.NewDeviceService(deviceRepo)
	pollingSvc := services.NewPollingService(deviceRepo, pollRepo, cfg.Polling, cfg.Alerting)
	authSvc := services.NewAuthService(userRepo, cfg.Auth)
	reportSvc := services.NewReportService(deviceRepo, pollRepo, pollingSvc, pollingSvc.Notifier())
	updateSvc := services.NewUpdateService(cfg.Updates)

	if cfg.Auth.Enabled {
		if cfg.Auth.JWTSecret == "" {
			logger.Fatal("auth.enabled is true but auth.jwt_secret is empty — set EMB_AUTH_JWT_SECRET or configs/config.yaml")
		}
		if err := seedAdmin(userRepo, cfg.Auth); err != nil {
			logger.Fatal("failed to seed admin user", zap.Error(err))
		}
	}

	pollingSvc.Start()
	pollingSvc.StartPruning()
	defer pollingSvc.Stop()

	// Background update checker (polls GitHub Releases; self-update is admin-triggered).
	updateCtx, updateCancel := context.WithCancel(context.Background())
	defer updateCancel()
	updateSvc.StartChecker(updateCtx)

	// Scheduled fleet report (delivers a text summary to the alerting webhook).
	if cfg.Reports.Enabled {
		c := cron.New()
		if _, err := c.AddFunc(cfg.Reports.Cron, reportSvc.DeliverScheduled); err != nil {
			logger.Fatal("invalid reports.cron expression", zap.String("cron", cfg.Reports.Cron), zap.Error(err))
		}
		c.Start()
		defer c.Stop()
		logger.Info("scheduled reports enabled", zap.String("cron", cfg.Reports.Cron))
	}

	router := api.NewRouter(cfg, deviceSvc, pollingSvc, pollRepo, authSvc, userRepo, reportSvc, updateSvc)

	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Bind with a short retry so a self-update relaunch can start before the old
	// process has fully released the port.
	ln, err := listenWithRetry(cfg.Server.Address(), 15*time.Second)
	if err != nil {
		logger.Fatal("failed to bind address", zap.String("address", cfg.Server.Address()), zap.Error(err))
	}
	go func() {
		logger.Info("server starting", zap.String("address", cfg.Server.Address()))
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
