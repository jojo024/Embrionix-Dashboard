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
	SlowResponseCount int    `gorm:"default:0" json:"slow_response_count"` // consecutive slow responses; resets on fast response
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

	// /self/diag/refclk - detailed PTP status
	PTP *PTPStatus `json:"ptp,omitempty"`

	// /self/firmware - firmware bank slots
	FirmwareSlots []FirmwareSlot `json:"firmware_slots,omitempty"`

	// /self/license - licensed feature map (feature -> "licensed"/"unlicensed")
	Licenses map[string]string `json:"licenses,omitempty"`

	// /self/diag/ethernet - control-plane packet counters
	Ethernet *EthernetStats `json:"ethernet,omitempty"`

	// /self/diag/common - device-level health stats
	VideoBandwidthUsage string `json:"video_bandwidth_usage,omitempty"`
	WatchdogStatus      string `json:"watchdog_status,omitempty"`
	IPv4PacketDrop      string `json:"ipv4_packet_drop,omitempty"`

	// /self/interfaces - per-interface network config (e1, e2 ...)
	Interfaces []NetworkInterface `json:"interfaces,omitempty"`

	// /lldp - discovered neighbour
	LLDP          *LLDPNeighbor  `json:"lldp,omitempty"`           // primary neighbour (first), kept for compatibility
	LLDPNeighbors []LLDPNeighbor `json:"lldp_neighbors,omitempty"` // all neighbours, one per local interface

	// /telemetry/devices - media flow packet counters
	MediaDevices []MediaDeviceTelemetry `json:"media_devices,omitempty"`

	// /sdi - SDI configuration / signal
	SDIBitRate string `json:"sdi_bit_rate,omitempty"`

	// /telemetry/ports - SFP per port
	Ports []PortTelemetry `json:"ports"`

	// Port SFP DDM details (from /port/{id})
	PortDetails []PortDetail `json:"port_details"`

	// Errors/alarms
	Alarms []string `json:"alarms"`
}

// PTPStatus holds decoded PTP/refclk diagnostics from /self/diag/refclk.
type PTPStatus struct {
	StatusCode       string `json:"status_code"`        // raw hex value e.g. "3"
	StatusLabel      string `json:"status_label"`       // human label: unlocked/coarse lock/locked
	Locked           bool   `json:"locked"`             // true when fully locked (status 3)
	MasterIP         string `json:"master_ip"`
	OffsetFromMaster int64  `json:"offset_from_master"` // nanoseconds
	MeanDelay        int64  `json:"mean_delay"`         // nanoseconds
	SyncCounter      int64  `json:"sync_counter"`
	DelayReqCounter  int64  `json:"delay_request_counter"`
	CoarseUnlock     bool   `json:"coarse_unlock"`
	Unlock           bool   `json:"unlock"`
}

// FirmwareSlot describes one firmware bank from /self/firmware.
type FirmwareSlot struct {
	Slot      int    `json:"slot"`
	ProductID int    `json:"product_id"`
	Desc      string `json:"desc"`
	Version   string `json:"version"`
	Active    bool   `json:"active"`
	Default   bool   `json:"default"`
}

// EthernetStats holds control-plane packet counters from /self/diag/ethernet.
type EthernetStats struct {
	TxPackets string `json:"tx_packets"`
	RxPackets string `json:"rx_packets"`
	RxError   string `json:"rx_error"`
	TxRate    string `json:"tx_rate"`
	RxRate    string `json:"rx_rate"`
}

// NetworkInterface describes one device interface (e1, e2) from /self/interfaces.
type NetworkInterface struct {
	Name           string `json:"name"`
	StaticIP       string `json:"static_ip"`
	StaticGateway  string `json:"static_gateway"`
	CurrentIP      string `json:"current_ip"`
	CurrentGateway string `json:"current_gateway"`
	DHCP           bool   `json:"dhcp"`
	VLAN           int    `json:"vlan"`
}

// LLDPNeighbor holds a discovered LLDP neighbour from /lldp. Interface is the
// local physical interface the advertisement was received on (matches the SFP
// port number on multi-port devices). The protocol exposes no neighbour
// hostname — chassis (a MAC) is the only identifier.
type LLDPNeighbor struct {
	Interface int    `json:"interface"`
	ChassisID string `json:"chassis_id"`
	PortID    string `json:"port_id"`
	TTL       string `json:"ttl"`
}

// MediaDeviceTelemetry summarises one media device's flow activity from /telemetry/devices.
type MediaDeviceTelemetry struct {
	Device   string `json:"device"`
	Channel  int    `json:"channel"`
	Type     string `json:"type"`
	Valid    bool   `json:"valid"`
	FlowCount int   `json:"flow_count"`
	TotalPkts int64 `json:"total_pkts"`
}

type PortTelemetry struct {
	Port        int      `json:"port"`
	Temperature float64  `json:"temperature"`
	TxPower     *float64 `json:"tx_power"`      // dBm (converted from µW in internal storage)
	RxPower     *float64 `json:"rx_power"`      // dBm (converted from µW in internal storage)
	TxPowerUW   int      `json:"tx_power_uw"`   // raw microwatts
	RxPowerUW   int      `json:"rx_power_uw"`   // raw microwatts
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
