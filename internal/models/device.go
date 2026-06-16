package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceStatus string

const (
	StatusUnknown  DeviceStatus = "unknown"
	StatusOnline   DeviceStatus = "online"
	StatusOffline  DeviceStatus = "offline"
	StatusWarning  DeviceStatus = "warning"
	StatusCritical DeviceStatus = "critical"
)

type Device struct {
	ID          string       `gorm:"primaryKey;type:text" json:"id"`
	Name        string       `gorm:"not null" json:"name"`
	Description string       `json:"description"`
	Location    string       `json:"location"`
	Rack        string       `json:"rack"`
	SerialNumber string      `json:"serial_number"`
	Model       string       `json:"model"`
	FirmwareVersion string   `json:"firmware_version"`
	ManagementIPRed  string  `gorm:"column:mgmt_ip_red" json:"management_ip_red"`
	ManagementIPBlue string  `gorm:"column:mgmt_ip_blue" json:"management_ip_blue"`
	Tags        string       `json:"tags"`
	Notes       string       `json:"notes"`
	MonitoringEnabled bool   `gorm:"default:true" json:"monitoring_enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`

	// Runtime status (not stored in DB, populated from polling)
	Status          DeviceStatus `gorm:"-" json:"status"`
	LastPolledAt    *time.Time   `gorm:"-" json:"last_polled_at"`
	ReachableRed    *bool        `gorm:"-" json:"reachable_red"`
	ReachableBlue   *bool        `gorm:"-" json:"reachable_blue"`
	PollingData     *DevicePollingData `gorm:"-" json:"polling_data,omitempty"`
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}

// DevicePollingData holds live data fetched from the device API.
type DevicePollingData struct {
	// /self/information
	CurrentVersion    string `json:"current_version"`
	EmsfpVersion      string `json:"emsfp_version"`
	DeviceType        string `json:"device_type"`
	PlatformHWVersion string `json:"platform_hw_version"`

	// /self/system
	CoreTemp    float64 `json:"core_temp"`
	FanSpeed    int     `json:"fan_speed"`
	CoreVoltage int     `json:"core_voltage"`
	Uptime      string  `json:"uptime"`

	// /self/ipconfig
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ip_addr"`
	DHCPEnable string `json:"dhcp_enable"`
	MACAddress string `json:"local_mac"`

	// /telemetry/node - refclk
	RefclkStatus   string `json:"refclk_status"`
	GrandmasterID  string `json:"grandmaster_id"`
	OffsetFromMaster int64 `json:"offset_from_master"`

	// /telemetry/ports - SFP per port
	Ports []PortTelemetry `json:"ports"`

	// Port SFP DDM details (from /port/{id})
	PortDetails []PortDetail `json:"port_details"`

	// Errors/alarms
	Alarms []string `json:"alarms"`
}

type PortTelemetry struct {
	Port        int     `json:"port"`
	Temperature float64 `json:"temperature"`
	TxPower     int     `json:"tx_power"`
	RxPower     int     `json:"rx_power"`
}

type PortDetail struct {
	PortID   string  `json:"port_id"`
	Link     string  `json:"link"`
	Speed    string  `json:"speed"`
	SFPType  string  `json:"sfp_type"`
	DDM      *SFPDDM `json:"ddm,omitempty"`
}

type SFPDDM struct {
	Temperature DDMValue      `json:"temperature"`
	VCC         DDMValue      `json:"vcc"`
	TxBias      DDMValue      `json:"tx_bias"`
	TxPower     DDMValue      `json:"tx_power"`
	RxPower     DDMValue      `json:"rx_power"`
	AlarmStatus DDMAlarmStatus `json:"alarm_status"`
	WarnStatus  DDMAlarmStatus `json:"warning_status"`
}

type DDMValue struct {
	HighAlarm   float64 `json:"high_alarm"`
	LowAlarm    float64 `json:"low_alarm"`
	HighWarning float64 `json:"high_warning"`
	LowWarning  float64 `json:"low_warning"`
	Current     float64 `json:"current"`
}

type DDMAlarmStatus struct {
	HighTemperature bool `json:"high_temperature"`
	LowTemperature  bool `json:"low_temperature"`
	HighVCC         bool `json:"high_vcc"`
	LowVCC          bool `json:"low_vcc"`
	HighTxBias      bool `json:"high_tx_bias"`
	LowTxBias       bool `json:"low_tx_bias"`
	HighTxPower     bool `json:"high_tx_power"`
	LowTxPower      bool `json:"low_tx_power"`
	HighRxPower     bool `json:"high_rx_power"`
	LowRxPower      bool `json:"low_rx_power"`
}
