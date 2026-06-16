package models

// DeviceConfig is a read-only snapshot of an EM6's configuration, fetched on
// demand (not part of the health poll). Every field maps to a documented GET
// endpoint under /emsfp/node/v1. No values here are written back to the device.
type DeviceConfig struct {
	Network      *NetworkConfig   `json:"network,omitempty"`       // /self/ipconfig
	System       *SystemConfig    `json:"system,omitempty"`        // /self/system
	Protocols    *ProtocolsConfig `json:"protocols,omitempty"`     // /self/protocols
	Syslog       *SyslogConfig    `json:"syslog,omitempty"`        // /self/syslog
	StaticRoutes []StaticRoute    `json:"static_routes,omitempty"` // /self/static_route
	DNS          *DNSConfig       `json:"dns,omitempty"`           // /self/diag/dns
}

type NetworkConfig struct {
	MACAddress    string `json:"mac_address"`
	IPAddress     string `json:"ip_addr"`
	SubnetMask    string `json:"subnet_mask"`
	Gateway       string `json:"gateway"`
	Hostname      string `json:"hostname"`
	Port          string `json:"port"`
	DHCPEnable    string `json:"dhcp_enable"`
	CtlVLANID     string `json:"ctl_vlan_id"`
	CtlVLANPCP    string `json:"ctl_vlan_pcp"`
	CtlVLANEnable string `json:"ctl_vlan_enable"`
}

type SystemConfig struct {
	StagingMode  int    `json:"staging_mode"`
	MinFanSpeed  int    `json:"min_fan_speed"`
	SMPTE2022_7  string `json:"smpte_2022_7_class"`
}

type ProtocolsConfig struct {
	MDNSEnable          string `json:"mdns_enable"`
	EmberServerPort     string `json:"ember_server_port"`
	SAPAnnouncementEnable string `json:"sap_announcement_enable"`
}

type SyslogConfig struct {
	Server     string                     `json:"server"`
	Port       int                        `json:"port"`
	Enable     bool                       `json:"enable"`
	Monitoring map[string]map[string]bool `json:"monitoring,omitempty"`
}

type StaticRoute struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
}

type DNSConfig struct {
	ServerAddress string `json:"server_address"`
	DomainName    string `json:"domain_name"`
}
