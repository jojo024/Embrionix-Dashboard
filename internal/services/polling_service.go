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
	LastPolledAt   *time.Time
	Reachable      bool
	ReachableRed   *bool
	ReachableBlue  *bool
	ResponseMs     int64
	ResponseMsRed  int64
	ResponseMsBlue int64
	Status         models.DeviceStatus
	Data           *models.DevicePollingData
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

// StartPruning launches a daily background job that deletes poll history older
// than the configured retention window. A retention of 0 disables pruning.
func (s *PollingService) StartPruning() {
	days := s.pollCfg.HistoryRetentionDays
	if days <= 0 {
		logger.Info("history pruning disabled (retention 0)")
		return
	}
	retention := time.Duration(days) * 24 * time.Hour
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.prune(retention) // prune once on startup
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.prune(retention)
			case <-s.stop:
				return
			}
		}
	}()
	logger.Info("history pruning started", zap.Int("retention_days", days))
}

func (s *PollingService) prune(retention time.Duration) {
	if err := s.pollRepo.PruneOlderThan(retention); err != nil {
		logger.Error("history pruning failed", zap.Error(err))
		return
	}
	logger.Debug("history pruned", zap.Duration("older_than", retention))
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
	if device.ManagementIPRed == "" && device.ManagementIPBlue == "" {
		return
	}

	timeout := time.Duration(s.pollCfg.TimeoutSeconds) * time.Second
	now := time.Now()

	state := &pollState{LastPolledAt: &now}
	pollResult := &models.PollResult{
		DeviceID: device.ID,
		PolledAt: now,
	}

	// Dual-path reachability: probe Red and Blue independently at L4.
	probeCtx, probeCancel := context.WithTimeout(context.Background(), timeout)
	defer probeCancel()
	if s.pollCfg.ICMPEnabled {
		s.probeDualPath(probeCtx, device, timeout, state, pollResult)
	}

	// Choose the IP for the full API poll: prefer a reachable path, else fall
	// back to Red (or Blue) so we still record a meaningful error.
	ip := device.ManagementIPRed
	if state.ReachableRed != nil && !*state.ReachableRed && device.ManagementIPBlue != "" {
		ip = device.ManagementIPBlue
	}
	if ip == "" {
		ip = device.ManagementIPBlue
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := NewEmsfpClient(ip, "80", s.pollCfg.TimeoutSeconds)

	start := time.Now()
	pollingData, err := client.Poll(ctx)
	responseMs := time.Since(start).Milliseconds()
	state.ResponseMs = responseMs
	pollResult.ResponseMs = responseMs

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

		state.Status = deriveStatus(pollingData)

		pollResult.Reachable = true
		temp := pollingData.CoreTemp
		fan := pollingData.FanSpeed
		volt := pollingData.CoreVoltage
		pollResult.CoreTemp = &temp
		pollResult.FanSpeed = &fan
		pollResult.CoreVoltage = &volt

		if pollingData.PTP != nil {
			locked := pollingData.PTP.Locked
			offset := pollingData.PTP.OffsetFromMaster
			pollResult.PTPLocked = &locked
			pollResult.PTPOffset = &offset
		}

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
	device.PollingData = state.Data

	// Prefer independent dual-path results when available; otherwise fall back
	// to the single API-poll reachability for the Red (primary) path.
	if state.ReachableRed != nil {
		device.ReachableRed = state.ReachableRed
	} else {
		b := state.Reachable
		device.ReachableRed = &b
	}
	device.ReachableBlue = state.ReachableBlue
}

// probeDualPath checks the Red and Blue management IPs independently at L4 and
// records the results on both the live state and the persisted poll result.
func (s *PollingService) probeDualPath(
	ctx context.Context,
	device models.Device,
	timeout time.Duration,
	state *pollState,
	pollResult *models.PollResult,
) {
	var wg sync.WaitGroup
	if device.ManagementIPRed != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, ms := ProbeTCP(ctx, device.ManagementIPRed, "80", timeout)
			state.ReachableRed = &ok
			state.ResponseMsRed = ms
			pollResult.ReachableRed = &ok
		}()
	}
	if device.ManagementIPBlue != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, ms := ProbeTCP(ctx, device.ManagementIPBlue, "80", timeout)
			state.ReachableBlue = &ok
			state.ResponseMsBlue = ms
			pollResult.ReachableBlue = &ok
		}()
	}
	wg.Wait()
}

// deriveStatus maps live polling data to a device status using all health
// signals collected from the EM6 (alarms, temperature, PTP lock, bandwidth).
func deriveStatus(pd *models.DevicePollingData) models.DeviceStatus {
	status := models.StatusOnline
	if len(pd.Alarms) > 0 {
		status = models.StatusWarning
	}

	// Critical conditions escalate above warning.
	if pd.CoreTemp > 75 {
		pd.Alarms = append(pd.Alarms, "Core temperature critical (>75°C)")
		status = models.StatusCritical
	}
	return status
}

// FleetAlarm is a single active alarm attributed to a device.
type FleetAlarm struct {
	DeviceID   string             `json:"device_id"`
	DeviceName string             `json:"device_name"`
	Status     models.DeviceStatus `json:"status"`
	Message    string             `json:"message"`
	PolledAt   *time.Time         `json:"polled_at"`
}

// FleetAlarms returns every active alarm across all monitored devices, plus an
// "unreachable" entry for devices whose latest poll failed. Device names are
// resolved from the supplied map (id -> name).
func (s *PollingService) FleetAlarms(names map[string]string) []FleetAlarm {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alarms []FleetAlarm
	for id, state := range s.results {
		name := names[id]
		if !state.Reachable {
			alarms = append(alarms, FleetAlarm{
				DeviceID:   id,
				DeviceName: name,
				Status:     models.StatusOffline,
				Message:    "Device unreachable",
				PolledAt:   state.LastPolledAt,
			})
			continue
		}
		if state.Data == nil {
			continue
		}
		for _, msg := range state.Data.Alarms {
			alarms = append(alarms, FleetAlarm{
				DeviceID:   id,
				DeviceName: name,
				Status:     state.Status,
				Message:    msg,
				PolledAt:   state.LastPolledAt,
			})
		}
	}
	return alarms
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
