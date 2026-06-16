package main

import (
	"context"
	"fmt"
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
	"go.uber.org/zap"
)

func main() {
	cfgPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(cfg.Logging.Level, cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&models.Device{}, &models.PollResult{}, &models.AppSetting{}, &models.AlertEvent{}); err != nil {
		logger.Fatal("failed to migrate database", zap.Error(err))
	}

	deviceRepo := repositories.NewDeviceRepository(db)
	pollRepo := repositories.NewPollRepository(db)
	deviceSvc := services.NewDeviceService(deviceRepo)
	pollingSvc := services.NewPollingService(deviceRepo, pollRepo, cfg.Polling, cfg.Alerting)

	pollingSvc.Start()
	pollingSvc.StartPruning()
	defer pollingSvc.Stop()

	router := api.NewRouter(cfg, deviceSvc, pollingSvc, pollRepo)

	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("address", cfg.Server.Address()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
