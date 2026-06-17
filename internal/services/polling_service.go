package services

import (
	"context"
	"fmt"
	"strings"
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
	alertCfg   config.AlertingConfig
	notifier   *Notifier

	mu      sync.RWMutex
	results map[string]*pollState // keyed by device ID

	cycle uint64 // incremented each pollAll; drives the full-vs-light decision

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
	alertCfg config.AlertingConfig,
) *PollingService {
	return &PollingService{
		deviceRepo: deviceRepo,
		pollRepo:   pollRepo,
		pollCfg:    cfg,
		alertCfg:   alertCfg,
		notifier:   NewNotifier(alertCfg.WebhookURL, alertCfg.WebhookOn),
		results:    make(map[string]*pollState),
		stop:       make(chan struct{}),
	}
}

// Notifier returns the alerting webhook notifier so other services (e.g. the
// scheduled report) can reuse the same configured destination.
func (s *PollingService) Notifier() *Notifier { return s.notifier }

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
	if err := s.pollRepo.PruneAlertsOlderThan(retention); err != nil {
		logger.Error("alert pruning failed", zap.Error(err))
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

	s.cycle++
	// Full poll on the first cycle, then every full_every cycles.
	fullEvery := s.pollCfg.FullEvery
	if fullEvery < 1 {
		fullEvery = 1
	}
	full := s.cycle == 1 || s.cycle%uint64(fullEvery) == 0

	// Bound concurrency so a large fleet doesn't burst the network at once.
	limit := s.pollCfg.MaxConcurrentPolls
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	for _, d := range devices {
		d := d
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.pollDevice(d, full)
		}()
	}
	wg.Wait()
}

func (s *PollingService) pollDevice(device models.Device, full bool) {
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

	// Carry forward the previous full-poll's static data on a light poll. If we
	// have never had a successful full poll, force a full one this cycle.
	var prevData *models.DevicePollingData
	if prev := s.GetState(device.ID); prev != nil {
		prevData = prev.Data
	}
	if prevData == nil {
		full = true
	}

	start := time.Now()
	pollingData, err := client.Poll(ctx, full, prevData)
	responseMs := time.Since(start).Milliseconds()
	state.ResponseMs = responseMs
	pollResult.ResponseMs = responseMs

	// Track consecutive slow responses. These devices typically respond in 2-3 seconds,
	// so threshold is 6 seconds or 75% of timeout, whichever is lower.
	// Mark as slow after 3 consecutive slow responses.
	const SlowThresholdMs = 6000 // 6 seconds
	slowThreshold := int64(SlowThresholdMs)
	if timeoutMs := int64(s.pollCfg.TimeoutSeconds) * 1000 * 3 / 4; timeoutMs < slowThreshold {
		slowThreshold = timeoutMs
	}

	if err == nil && responseMs > slowThreshold {
		device.SlowResponseCount++
	} else if err == nil {
		// Reset on a fast response
		device.SlowResponseCount = 0
	}
	// On error, keep the slow count (error = effectively slow)

	// Update slow response count in DB
	if err := s.deviceRepo.Update(&device); err != nil {
		logger.Warn("failed to update device slow response count", zap.Error(err))
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

		state.Status = s.deriveStatus(pollingData, responseMs)

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
	prev := s.results[device.ID]
	s.results[device.ID] = state
	s.mu.Unlock()

	// Detect a status transition and record/notify. The first poll (no prior
	// state) and the unknown->X warm-up are not treated as alertable events.
	if prev != nil && prev.Status != "" && prev.Status != state.Status {
		s.handleTransition(device, prev.Status, state.Status)
	}

	if err := s.pollRepo.Save(pollResult); err != nil {
		logger.Error("failed to save poll result", zap.Error(err))
	}
}

// handleTransition records a status change as an AlertEvent and fires a webhook
// when the destination status is configured for notification.
func (s *PollingService) handleTransition(device models.Device, from, to models.DeviceStatus) {
	event := models.AlertEvent{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		FromStatus: from,
		ToStatus:   to,
		Message:    transitionMessage(device, to),
		CreatedAt:  time.Now(),
	}

	if err := s.pollRepo.SaveAlert(&event); err != nil {
		logger.Error("failed to save alert event", zap.Error(err))
	}
	logger.Info("device status transition",
		zap.String("device", device.Name),
		zap.String("from", string(from)),
		zap.String("to", string(to)),
	)

	if s.notifier.ShouldNotify(to) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			s.notifier.Notify(ctx, event)
		}()
	}
}

// transitionMessage produces a human summary for an alert event, surfacing the
// device's active alarms where relevant.
func transitionMessage(device models.Device, to models.DeviceStatus) string {
	switch to {
	case models.StatusOffline:
		return "Device became unreachable"
	case models.StatusOnline:
		return "Device recovered to healthy"
	default:
		return fmt.Sprintf("Device is now %s", to)
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
			// The Blue interface on an EM6 typically answers ICMP but does not run
			// the HTTP management server, so a TCP probe would falsely read offline.
			// Default to ICMP for Blue; allow TCP via polling.blue_probe.
			var ok bool
			var ms int64
			if strings.EqualFold(s.pollCfg.BlueProbe, "tcp") {
				ok, ms = ProbeTCP(ctx, device.ManagementIPBlue, "80", timeout)
			} else {
				ok, ms = ProbeICMP(ctx, device.ManagementIPBlue, timeout)
			}
			state.ReachableBlue = &ok
			state.ResponseMsBlue = ms
			pollResult.ReachableBlue = &ok
		}()
	}
	wg.Wait()
}

// deriveStatus maps live polling data to a device status using all health
// signals collected from the EM6 (alarms, temperature, PTP lock, bandwidth)
// against the configured alert thresholds.
func (s *PollingService) deriveStatus(pd *models.DevicePollingData, responseMs int64) models.DeviceStatus {
	a := s.alertCfg
	status := models.StatusOnline
	if len(pd.Alarms) > 0 {
		status = models.StatusWarning
	}

	// Warning-level threshold checks.
	if a.TempWarningC > 0 && pd.CoreTemp >= a.TempWarningC && pd.CoreTemp < a.TempCriticalC {
		pd.Alarms = append(pd.Alarms, fmt.Sprintf("Core temperature high (≥%.0f°C)", a.TempWarningC))
		if status == models.StatusOnline {
			status = models.StatusWarning
		}
	}
	if a.ResponseWarnMs > 0 && responseMs >= a.ResponseWarnMs {
		pd.Alarms = append(pd.Alarms, fmt.Sprintf("Slow API response (%dms)", responseMs))
		if status == models.StatusOnline {
			status = models.StatusWarning
		}
	}

	// Critical conditions escalate above warning.
	if a.TempCriticalC > 0 && pd.CoreTemp >= a.TempCriticalC {
		pd.Alarms = append(pd.Alarms, fmt.Sprintf("Core temperature critical (≥%.0f°C)", a.TempCriticalC))
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
