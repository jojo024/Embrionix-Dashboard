package services

import (
	"context"
	"sync"
	"time"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/embrionix/dashboard/pkg/logger"
	"go.uber.org/zap"
)

// PollingService runs background polls against all monitored devices.
type PollingService struct {
	deviceRepo *repositories.DeviceRepository
	pollRepo   *repositories.PollRepository
	pollCfg    config.PollingConfig

	mu      sync.RWMutex
	results map[string]*pollState // keyed by device ID

	stop chan struct{}
	wg   sync.WaitGroup
}

type pollState struct {
	LastPolledAt *time.Time
	Reachable    bool
	ResponseMs   int64
	Status       models.DeviceStatus
	Data         *models.DevicePollingData
}

func NewPollingService(
	deviceRepo *repositories.DeviceRepository,
	pollRepo *repositories.PollRepository,
	cfg config.PollingConfig,
) *PollingService {
	return &PollingService{
		deviceRepo: deviceRepo,
		pollRepo:   pollRepo,
		pollCfg:    cfg,
		results:    make(map[string]*pollState),
		stop:       make(chan struct{}),
	}
}

func (s *PollingService) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.pollAll() // initial poll on startup
		ticker := time.NewTicker(time.Duration(s.pollCfg.IntervalSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.pollAll()
			case <-s.stop:
				return
			}
		}
	}()
	logger.Info("polling service started", zap.Int("interval_seconds", s.pollCfg.IntervalSeconds))
}

func (s *PollingService) Stop() {
	close(s.stop)
	s.wg.Wait()
	logger.Info("polling service stopped")
}

func (s *PollingService) pollAll() {
	devices, err := s.deviceRepo.FindMonitoringEnabled()
	if err != nil {
		logger.Error("failed to load devices for polling", zap.Error(err))
		return
	}

	var wg sync.WaitGroup
	for _, d := range devices {
		d := d
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.pollDevice(d)
		}()
	}
	wg.Wait()
}

func (s *PollingService) pollDevice(device models.Device) {
	ip := device.ManagementIPRed
	if ip == "" {
		ip = device.ManagementIPBlue
	}
	if ip == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.pollCfg.TimeoutSeconds)*time.Second)
	defer cancel()

	client := NewEmsfpClient(ip, "80", s.pollCfg.TimeoutSeconds)

	start := time.Now()
	pollingData, err := client.Poll(ctx)
	responseMs := time.Since(start).Milliseconds()
	now := time.Now()

	state := &pollState{
		LastPolledAt: &now,
		ResponseMs:   responseMs,
	}

	pollResult := &models.PollResult{
		DeviceID:   device.ID,
		PolledAt:   now,
		ResponseMs: responseMs,
	}

	if err != nil {
		state.Reachable = false
		state.Status = models.StatusOffline
		pollResult.Reachable = false
		pollResult.ErrorMessage = err.Error()
		logger.Warn("device poll failed",
			zap.String("device", device.Name),
			zap.String("ip", ip),
			zap.Error(err),
		)
	} else {
		state.Reachable = true
		state.Data = pollingData

		// Determine status based on alarms
		if len(pollingData.Alarms) > 0 {
			state.Status = models.StatusWarning
		} else {
			state.Status = models.StatusOnline
		}

		// Temperature critical threshold: >75°C
		if pollingData.CoreTemp > 75 {
			state.Status = models.StatusCritical
			pollingData.Alarms = append(pollingData.Alarms, "Core temperature critical")
		}

		pollResult.Reachable = true
		temp := pollingData.CoreTemp
		fan := pollingData.FanSpeed
		volt := pollingData.CoreVoltage
		pollResult.CoreTemp = &temp
		pollResult.FanSpeed = &fan
		pollResult.CoreVoltage = &volt

		if len(pollingData.Ports) > 0 {
			p0tx := pollingData.Ports[0].TxPower
			p0rx := pollingData.Ports[0].RxPower
			p0t := pollingData.Ports[0].Temperature
			pollResult.Port0TxPower = &p0tx
			pollResult.Port0RxPower = &p0rx
			pollResult.Port0Temp = &p0t
		}
		if len(pollingData.Ports) > 1 {
			p1tx := pollingData.Ports[1].TxPower
			p1rx := pollingData.Ports[1].RxPower
			p1t := pollingData.Ports[1].Temperature
			pollResult.Port1TxPower = &p1tx
			pollResult.Port1RxPower = &p1rx
			pollResult.Port1Temp = &p1t
		}

		logger.Debug("device polled successfully",
			zap.String("device", device.Name),
			zap.String("ip", ip),
			zap.Int64("response_ms", responseMs),
		)
	}

	s.mu.Lock()
	s.results[device.ID] = state
	s.mu.Unlock()

	if err := s.pollRepo.Save(pollResult); err != nil {
		logger.Error("failed to save poll result", zap.Error(err))
	}
}

// GetState returns the latest poll state for a device.
func (s *PollingService) GetState(deviceID string) *pollState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[deviceID]
}

// EnrichDevice attaches live polling data to a device record.
func (s *PollingService) EnrichDevice(device *models.Device) {
	state := s.GetState(device.ID)
	if state == nil {
		device.Status = models.StatusUnknown
		return
	}
	device.Status = state.Status
	device.LastPolledAt = state.LastPolledAt
	b := state.Reachable
	device.ReachableRed = &b
	device.PollingData = state.Data
}

// Summary returns aggregate counts across all devices.
func (s *PollingService) Summary() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]int{
		"online": 0, "offline": 0, "warning": 0, "critical": 0, "unknown": 0,
	}
	for _, state := range s.results {
		switch state.Status {
		case models.StatusOnline:
			counts["online"]++
		case models.StatusOffline:
			counts["offline"]++
		case models.StatusWarning:
			counts["warning"]++
		case models.StatusCritical:
			counts["critical"]++
		default:
			counts["unknown"]++
		}
	}
	return counts
}
