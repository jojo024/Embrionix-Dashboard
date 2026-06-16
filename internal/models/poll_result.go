package models

import (
	"time"
)

// PollResult stores historical polling data for a device.
type PollResult struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID    string    `gorm:"index;not null" json:"device_id"`
	PolledAt    time.Time `gorm:"index" json:"polled_at"`
	Reachable   bool      `json:"reachable"`
	ResponseMs  int64     `json:"response_ms"`

	// Snapshot of key metrics
	CoreTemp    *float64 `json:"core_temp"`
	FanSpeed    *int     `json:"fan_speed"`
	CoreVoltage *int     `json:"core_voltage"`

	// Port 0 SFP (primary)
	Port0TxPower *int     `json:"port0_tx_power"`
	Port0RxPower *int     `json:"port0_rx_power"`
	Port0Temp    *float64 `json:"port0_temp"`

	// Port 1 SFP (secondary)
	Port1TxPower *int     `json:"port1_tx_power"`
	Port1RxPower *int     `json:"port1_rx_power"`
	Port1Temp    *float64 `json:"port1_temp"`

	// PTP / refclk trend
	PTPLocked *bool  `json:"ptp_locked"`
	PTPOffset *int64 `json:"ptp_offset"`

	// Dual-path reachability snapshot
	ReachableRed  *bool `json:"reachable_red"`
	ReachableBlue *bool `json:"reachable_blue"`

	ErrorMessage string `json:"error_message,omitempty"`
}

// AppSetting stores key/value application configuration.
type AppSetting struct {
	Key       string    `gorm:"primaryKey;type:text" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertEvent records a device status transition (e.g. online -> critical) for
// the alert history and is the payload sent to notification webhooks.
type AlertEvent struct {
	ID         uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID   string       `gorm:"index;not null" json:"device_id"`
	DeviceName string       `json:"device_name"`
	FromStatus DeviceStatus `json:"from_status"`
	ToStatus   DeviceStatus `json:"to_status"`
	Message    string       `json:"message"`
	CreatedAt  time.Time    `gorm:"index" json:"created_at"`
}
