package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/embrionix/dashboard/internal/models"
)

// EmsfpClient communicates with a single Embrionix EM6 device via its REST API.
// Base URL: http://<ip>/emsfp/node/v1
type EmsfpClient struct {
	httpClient *http.Client
	baseURL    string
}

// DeviceInfo holds basic identifying info fetched from a device.
type DeviceInfo struct {
	CurrentVersion string
}

func NewEmsfpClient(ip string, port string, timeoutSec int) *EmsfpClient {
	if port == "" || port == "80" {
		port = "80"
	}
	return &EmsfpClient{
		baseURL: fmt.Sprintf("http://%s:%s/emsfp/node/v1", ip, port),
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(timeoutSec) * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				// Reuse one TCP connection for the many GETs in a single poll
				// (the device's embedded HTTP server is the constrained party),
				// but let it close shortly after so we don't tie up the device's
				// limited connection slots between cycles.
				DisableKeepAlives:   false,
				MaxIdleConns:        2,
				MaxIdleConnsPerHost: 2,
				MaxConnsPerHost:     2,
				IdleConnTimeout:     5 * time.Second,
			},
		},
	}
}

// FetchInfo quickly fetches the device's current firmware version.
func (c *EmsfpClient) FetchInfo(ctx context.Context) (*DeviceInfo, error) {
	var info selfInformation
	if err := c.get(ctx, "/self/information", &info); err != nil {
		return nil, err
	}
	return &DeviceInfo{
		CurrentVersion: info.CurrentVersion,
	}, nil
}

func (c *EmsfpClient) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// put sends a JSON PUT to the device. The body is marshalled from in; the
// response body is discarded (device PUTs return a status envelope). A non-2xx
// status is returned as an error so callers can surface and audit the failure.
func (c *EmsfpClient) put(ctx context.Context, path string, in interface{}) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, path, string(msg))
	}
	return nil
}

// --- Response structs mirroring the emSFP API ---

type selfInformation struct {
	CurrentVersion    string `json:"current_version"`
	EmsfpVersion      string `json:"emsfp_version"`
	AsicVersion       string `json:"asic_version"`
	Type              string `json:"type"`
	PlatformHWVersion string `json:"platform_hw_version"`
	BaseHWVersion     string `json:"base_hw_version"`
}

type selfIPConfig struct {
	Version    string `json:"version"`
	LocalMAC   string `json:"local_mac"`
	IPAddr     string `json:"ip_addr"`
	SubnetMask string `json:"subnet_mask"`
	Gateway    string `json:"gateway"`
	Hostname   string `json:"hostname"`
	Port       string `json:"port"`
	DHCPEnable string `json:"dhcp_enable"`
}

type selfSystem struct {
	Reboot      string  `json:"reboot"`
	CoreTemp    float64 `json:"core_temp"`
	CoreVoltage int     `json:"core_voltage"`
	Uptime      string  `json:"uptime"`
	FanSpeed    int     `json:"fan_speed"`
}

type telemetryNode struct {
	Health struct {
		CoreTemp    float64 `json:"core_temp"`
		FanSpeed    int     `json:"fan_speed"`
		CoreVoltage int     `json:"core_voltage"`
	} `json:"health"`
	Refclk struct {
		Status           string `json:"status"`
		GrandmasterID    string `json:"grandmaster_id"`
		OffsetFromMaster int64  `json:"offset_from_master"`
		MeanDelay        int64  `json:"mean_delay"`
	} `json:"refclk"`
}

type telemetryPorts struct {
	Ports []struct {
		Port        int     `json:"port"`
		Temperature float64 `json:"temperature"`
		TxPower     int     `json:"tx_power"`
		RxPower     int     `json:"rx_power"`
	} `json:"ports"`
}

type portInfo struct {
	Link     string `json:"link"`
	Speed    string `json:"speed"`
	SFPType  string `json:"sfp_type"`
	SFPDDMInfo *struct {
		Temperature struct {
			HighAlarm   float64 `json:"high_alarm"`
			LowAlarm    float64 `json:"low_alarm"`
			HighWarning float64 `json:"high_warning"`
			LowWarning  float64 `json:"low_warning"`
			Current     float64 `json:"current"`
		} `json:"temperature"`
		VCC struct {
			Current float64 `json:"current"`
		} `json:"vcc"`
		TxBias struct {
			Current float64 `json:"current"`
		} `json:"tx_bias"`
		TxPower struct {
			Current float64 `json:"current"`
		} `json:"tx_power"`
		RxPower struct {
			Current float64 `json:"current"`
		} `json:"rx_power"`
		AlarmStatus struct {
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
		} `json:"alarm_status"`
		WarningStatus struct {
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
		} `json:"warning_status"`
	} `json:"sfp_ddm_info"`
}

// --- Additional monitoring endpoint structs ---

type selfFirmware struct {
	Info []struct {
		Slot      int    `json:"slot"`
		ProductID int    `json:"product_id"`
		Desc      string `json:"desc"`
		Version   string `json:"version"`
		Active    string `json:"active"`
		Default   string `json:"default"`
	} `json:"info"`
}

type selfLicense struct {
	Feature map[string]string `json:"feature"`
}

type selfDiagEthernet struct {
	Stats struct {
		TxPackets string `json:"tx_packets"`
		RxPackets string `json:"rx_packets"`
		RxError   string `json:"rx_error"`
		TxRate    string `json:"tx_rate"`
		RxRate    string `json:"rx_rate"`
	} `json:"stats"`
}

type selfDiagRefclk struct {
	Status           string `json:"status"`
	RefclkMasterIP   string `json:"refclk_master_ip"`
	OffsetFromMaster int64  `json:"offset_from_master"`
	MeanDelay        int64  `json:"mean_delay"`
	Events           struct {
		CoarseUnlock bool `json:"coarse_unlock"`
		Unlock       bool `json:"unlock"`
	} `json:"events"`
	Counters struct {
		SyncCounter         int64 `json:"sync_counter"`
		DelayRequestCounter int64 `json:"delay_request_counter"`
	} `json:"counters"`
}

type selfDiagCommon struct {
	Stats struct {
		IPv4PacketDrop      string `json:"ipv4_packet_drop"`
		WatchdogStatus      string `json:"watchdog_status"`
		VideoBandwidthUsage string `json:"video_bandwidth_usage"`
	} `json:"stats"`
}

type selfInterfaceEntry struct {
	StaticIP       string `json:"static_ip"`
	StaticGateway  string `json:"static_gateway"`
	CurrentIP      string `json:"current_ip"`
	CurrentGateway string `json:"current_gateway"`
	DHCP           bool   `json:"dhcp"`
	VLAN           int    `json:"vlan"`
}

type lldpResponse struct {
	Neighbor struct {
		Chassis string `json:"chassis"`
		Port    string `json:"port"`
		TTL     string `json:"ttl"`
	} `json:"neighbor"`
}

type telemetryDevices struct {
	Devices []struct {
		Device  string `json:"device"`
		Channel int    `json:"channel"`
		Valid   bool   `json:"valid"`
		Type    string `json:"type"`
		Engines []struct {
			Flows []struct {
				PktCnt int64 `json:"pkt_cnt"`
			} `json:"flows"`
		} `json:"engines"`
	} `json:"devices"`
}

type sdiResponse struct {
	Configuration struct {
		OperatingBitRate string `json:"operating_bit_rate"`
	} `json:"configuration"`
}

// ptpStatusLabel decodes the EM6 refclk status hex code into a human label.
// Per the EM6 API: 0 = not locked, 1 = stage 1 (coarse) lock, 3 = stage 2 (locked).
func ptpStatusLabel(code string) (label string, locked bool) {
	switch code {
	case "0", "0x0", "":
		return "unlocked", false
	case "1", "0x1":
		return "coarse lock", false
	case "3", "0x3":
		return "locked", true
	default:
		return "code " + code, false
	}
}

// Poll fetches device data and returns a DevicePollingData.
//
// To minimise load on the device's embedded HTTP server, polling is tiered:
//   - The "fast" tier (dynamic health: temp, fan, SFP power, PTP, bandwidth) is
//     fetched on every call — ~6 requests.
//   - The "slow" tier (static or heavy data: ipconfig, firmware, license,
//     ethernet counters, interfaces, LLDP, media flows, SDI, and per-port SFP
//     DDM detail) is fetched only when full is true; otherwise those fields are
//     carried over from prev. This drops steady-state requests from ~17 to ~6.
//
// Alarms are derived once at the end from the final (fast + carried/fresh slow)
// data, so they remain correct on light polls.
func (c *EmsfpClient) Poll(ctx context.Context, full bool, prev *models.DevicePollingData) (*models.DevicePollingData, error) {
	data := &models.DevicePollingData{}

	// --- Fast tier (always) ---

	// /self/information — also the reachability check, so a failure aborts.
	var info selfInformation
	if err := c.get(ctx, "/self/information", &info); err != nil {
		return nil, fmt.Errorf("self/information: %w", err)
	}
	data.CurrentVersion = info.CurrentVersion
	data.EmsfpVersion = info.EmsfpVersion
	data.DeviceType = info.Type
	data.PlatformHWVersion = info.PlatformHWVersion

	var sys selfSystem
	if err := c.get(ctx, "/self/system", &sys); err == nil {
		data.CoreTemp = sys.CoreTemp
		data.FanSpeed = sys.FanSpeed
		data.CoreVoltage = sys.CoreVoltage
		data.Uptime = sys.Uptime
	}

	var telNode telemetryNode
	if err := c.get(ctx, "/telemetry/node", &telNode); err == nil {
		if telNode.Health.CoreTemp > 0 {
			data.CoreTemp = telNode.Health.CoreTemp
			data.FanSpeed = telNode.Health.FanSpeed
			data.CoreVoltage = telNode.Health.CoreVoltage
		}
		data.RefclkStatus = telNode.Refclk.Status
		data.GrandmasterID = telNode.Refclk.GrandmasterID
		data.OffsetFromMaster = telNode.Refclk.OffsetFromMaster
	}

	var telPorts telemetryPorts
	if err := c.get(ctx, "/telemetry/ports", &telPorts); err == nil {
		for _, p := range telPorts.Ports {
			data.Ports = append(data.Ports, models.PortTelemetry{
				Port:        p.Port,
				Temperature: p.Temperature,
				TxPower:     p.TxPower,
				RxPower:     p.RxPower,
			})
		}
	}

	var refclk selfDiagRefclk
	if err := c.get(ctx, "/self/diag/refclk", &refclk); err == nil {
		label, locked := ptpStatusLabel(refclk.Status)
		data.PTP = &models.PTPStatus{
			StatusCode:       refclk.Status,
			StatusLabel:      label,
			Locked:           locked,
			MasterIP:         refclk.RefclkMasterIP,
			OffsetFromMaster: refclk.OffsetFromMaster,
			MeanDelay:        refclk.MeanDelay,
			SyncCounter:      refclk.Counters.SyncCounter,
			DelayReqCounter:  refclk.Counters.DelayRequestCounter,
			CoarseUnlock:     refclk.Events.CoarseUnlock,
			Unlock:           refclk.Events.Unlock,
		}
		if data.RefclkStatus == "" {
			data.RefclkStatus = label
		}
		if data.OffsetFromMaster == 0 {
			data.OffsetFromMaster = refclk.OffsetFromMaster
		}
	}

	var common selfDiagCommon
	if err := c.get(ctx, "/self/diag/common", &common); err == nil {
		data.VideoBandwidthUsage = common.Stats.VideoBandwidthUsage
		data.WatchdogStatus = common.Stats.WatchdogStatus
		data.IPv4PacketDrop = common.Stats.IPv4PacketDrop
	}

	// --- Slow tier (only on a full poll; otherwise carry forward) ---
	if full {
		c.pollSlow(ctx, data)
	} else if prev != nil {
		carrySlowData(data, prev)
	}

	buildAlarms(data)
	return data, nil
}

// pollSlow fetches the static / heavy endpoints into data.
func (c *EmsfpClient) pollSlow(ctx context.Context, data *models.DevicePollingData) {
	var ipcfg selfIPConfig
	if err := c.get(ctx, "/self/ipconfig", &ipcfg); err == nil {
		data.Hostname = ipcfg.Hostname
		data.IPAddress = ipcfg.IPAddr
		data.DHCPEnable = ipcfg.DHCPEnable
		data.MACAddress = ipcfg.LocalMAC
	}

	var fw selfFirmware
	if err := c.get(ctx, "/self/firmware", &fw); err == nil {
		for _, s := range fw.Info {
			if s.ProductID == 0 && s.Version == "" {
				continue
			}
			data.FirmwareSlots = append(data.FirmwareSlots, models.FirmwareSlot{
				Slot:      s.Slot,
				ProductID: s.ProductID,
				Desc:      strings.TrimSpace(s.Desc),
				Version:   s.Version,
				Active:    s.Active == "yes",
				Default:   s.Default == "yes",
			})
		}
	}

	var lic selfLicense
	if err := c.get(ctx, "/self/license", &lic); err == nil && len(lic.Feature) > 0 {
		data.Licenses = lic.Feature
	}

	var eth selfDiagEthernet
	if err := c.get(ctx, "/self/diag/ethernet", &eth); err == nil {
		data.Ethernet = &models.EthernetStats{
			TxPackets: eth.Stats.TxPackets,
			RxPackets: eth.Stats.RxPackets,
			RxError:   eth.Stats.RxError,
			TxRate:    eth.Stats.TxRate,
			RxRate:    eth.Stats.RxRate,
		}
	}

	var ifaces map[string]selfInterfaceEntry
	if err := c.get(ctx, "/self/interfaces", &ifaces); err == nil {
		names := make([]string, 0, len(ifaces))
		for name := range ifaces {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			e := ifaces[name]
			data.Interfaces = append(data.Interfaces, models.NetworkInterface{
				Name:           name,
				StaticIP:       e.StaticIP,
				StaticGateway:  e.StaticGateway,
				CurrentIP:      e.CurrentIP,
				CurrentGateway: e.CurrentGateway,
				DHCP:           e.DHCP,
				VLAN:           e.VLAN,
			})
		}
	}

	var lldp lldpResponse
	if err := c.get(ctx, "/lldp", &lldp); err == nil && lldp.Neighbor.Chassis != "" {
		data.LLDP = &models.LLDPNeighbor{
			ChassisID: lldp.Neighbor.Chassis,
			PortID:    lldp.Neighbor.Port,
			TTL:       lldp.Neighbor.TTL,
		}
	}

	var telDevices telemetryDevices
	if err := c.get(ctx, "/telemetry/devices", &telDevices); err == nil {
		for _, d := range telDevices.Devices {
			md := models.MediaDeviceTelemetry{
				Device:  d.Device,
				Channel: d.Channel,
				Type:    d.Type,
				Valid:   d.Valid,
			}
			for _, eng := range d.Engines {
				for _, fl := range eng.Flows {
					md.FlowCount++
					md.TotalPkts += fl.PktCnt
				}
			}
			data.MediaDevices = append(data.MediaDevices, md)
		}
	}

	var sdi sdiResponse
	if err := c.get(ctx, "/sdi", &sdi); err == nil {
		data.SDIBitRate = sdi.Configuration.OperatingBitRate
	}

	// Per-port SFP DDM detail — the heaviest part (one request per port).
	portListRaw := []string{}
	if err := c.get(ctx, "/port", &portListRaw); err == nil {
		for _, portEntry := range portListRaw {
			portID := portEntry
			if len(portID) > 0 && portID[len(portID)-1] == '/' {
				portID = portID[:len(portID)-1]
			}
			var pi portInfo
			if err := c.get(ctx, "/port/"+portID, &pi); err == nil {
				pd := models.PortDetail{PortID: portID, Link: pi.Link, Speed: pi.Speed, SFPType: pi.SFPType}
				if pi.SFPDDMInfo != nil {
					ddm := &models.SFPDDM{}
					ddm.Temperature = models.DDMValue{
						HighAlarm:   pi.SFPDDMInfo.Temperature.HighAlarm,
						LowAlarm:    pi.SFPDDMInfo.Temperature.LowAlarm,
						HighWarning: pi.SFPDDMInfo.Temperature.HighWarning,
						LowWarning:  pi.SFPDDMInfo.Temperature.LowWarning,
						Current:     pi.SFPDDMInfo.Temperature.Current,
					}
					ddm.VCC = models.DDMValue{Current: pi.SFPDDMInfo.VCC.Current}
					ddm.TxBias = models.DDMValue{Current: pi.SFPDDMInfo.TxBias.Current}
					ddm.TxPower = models.DDMValue{Current: pi.SFPDDMInfo.TxPower.Current}
					ddm.RxPower = models.DDMValue{Current: pi.SFPDDMInfo.RxPower.Current}
					ddm.AlarmStatus = models.DDMAlarmStatus{
						HighTemperature: pi.SFPDDMInfo.AlarmStatus.HighTemperature,
						LowTemperature:  pi.SFPDDMInfo.AlarmStatus.LowTemperature,
						HighVCC:         pi.SFPDDMInfo.AlarmStatus.HighVCC,
						LowVCC:          pi.SFPDDMInfo.AlarmStatus.LowVCC,
						HighTxBias:      pi.SFPDDMInfo.AlarmStatus.HighTxBias,
						LowTxBias:       pi.SFPDDMInfo.AlarmStatus.LowTxBias,
						HighTxPower:     pi.SFPDDMInfo.AlarmStatus.HighTxPower,
						LowTxPower:      pi.SFPDDMInfo.AlarmStatus.LowTxPower,
						HighRxPower:     pi.SFPDDMInfo.AlarmStatus.HighRxPower,
						LowRxPower:      pi.SFPDDMInfo.AlarmStatus.LowRxPower,
					}
					pd.DDM = ddm
				}
				data.PortDetails = append(data.PortDetails, pd)
			}
		}
	}
}

// carrySlowData copies the static / heavy fields from a previous full poll onto
// a light-poll result so the UI keeps showing them between full polls.
func carrySlowData(data, prev *models.DevicePollingData) {
	data.Hostname = prev.Hostname
	data.IPAddress = prev.IPAddress
	data.DHCPEnable = prev.DHCPEnable
	data.MACAddress = prev.MACAddress
	data.FirmwareSlots = prev.FirmwareSlots
	data.Licenses = prev.Licenses
	data.Ethernet = prev.Ethernet
	data.Interfaces = prev.Interfaces
	data.LLDP = prev.LLDP
	data.MediaDevices = prev.MediaDevices
	data.SDIBitRate = prev.SDIBitRate
	data.PortDetails = prev.PortDetails
}

// buildAlarms derives the active-alarm list from the final polling data.
func buildAlarms(data *models.DevicePollingData) {
	if data.PTP != nil && !data.PTP.Locked {
		data.Alarms = append(data.Alarms, fmt.Sprintf("PTP not locked (%s)", data.PTP.StatusLabel))
	}
	if data.Ethernet != nil {
		if e := data.Ethernet.RxError; e != "" && e != "N/A" && e != "0" {
			data.Alarms = append(data.Alarms, "Ethernet RX errors detected: "+e)
		}
	}
	if v := data.VideoBandwidthUsage; v != "" && v != "good" && v != "N/A" {
		data.Alarms = append(data.Alarms, "Video bandwidth usage: "+v)
	}
	for _, pd := range data.PortDetails {
		if pd.DDM == nil {
			continue
		}
		a := pd.DDM.AlarmStatus
		if a.HighTemperature {
			data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: High temperature alarm", pd.PortID))
		}
		if a.HighRxPower {
			data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: High RX power alarm", pd.PortID))
		}
		if a.LowRxPower {
			data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: Low RX power alarm", pd.PortID))
		}
		if a.LowTxPower {
			data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: Low TX power alarm", pd.PortID))
		}
	}
}

// --- Read-only configuration endpoint structs ---

type selfSystemConfig struct {
	StagingMode int `json:"staging_mode"`
	MinFanSpeed int `json:"min_fan_speed"`
	SMPTENetwork struct {
		ST20227 struct {
			Class string `json:"class"`
		} `json:"2022-7"`
	} `json:"smpte_network"`
}

type selfSyslog struct {
	Config struct {
		Server string `json:"server"`
		Port   int    `json:"port"`
		Enable bool   `json:"enable"`
	} `json:"config"`
	Monitoring map[string]map[string]bool `json:"monitoring"`
}

type selfDiagDNS struct {
	DNS struct {
		ServerAddress string `json:"server_address"`
		DomainName    string `json:"domain_name"`
	} `json:"dns"`
}

type selfProtocols struct {
	MDNSEnable            string `json:"mdns_enable"`
	EmberServerPort       string `json:"ember_server_port"`
	SAPAnnouncementEnable string `json:"sap_announcement_enable"`
}

// FetchConfig retrieves the device's read-only configuration on demand. Like
// Poll, every endpoint is best-effort: an endpoint the device type does not
// implement is skipped rather than failing the whole request. This method
// performs GETs only — it never writes to the device.
func (c *EmsfpClient) FetchConfig(ctx context.Context) (*models.DeviceConfig, error) {
	cfg := &models.DeviceConfig{}

	var ipcfg selfIPConfig
	if err := c.get(ctx, "/self/ipconfig", &ipcfg); err != nil {
		// ipconfig is the baseline; if it fails the device is unreachable.
		return nil, fmt.Errorf("self/ipconfig: %w", err)
	}
	cfg.Network = &models.NetworkConfig{
		MACAddress: ipcfg.LocalMAC,
		IPAddress:  ipcfg.IPAddr,
		SubnetMask: ipcfg.SubnetMask,
		Gateway:    ipcfg.Gateway,
		Hostname:   ipcfg.Hostname,
		Port:       ipcfg.Port,
		DHCPEnable: ipcfg.DHCPEnable,
	}

	// Re-read ipconfig into a map for the VLAN fields not on selfIPConfig.
	var ipRaw map[string]interface{}
	if err := c.get(ctx, "/self/ipconfig", &ipRaw); err == nil {
		cfg.Network.CtlVLANID = asString(ipRaw["ctl_vlan_id"])
		cfg.Network.CtlVLANPCP = asString(ipRaw["ctl_vlan_pcp"])
		cfg.Network.CtlVLANEnable = asString(ipRaw["ctl_vlan_enable"])
	}

	var sys selfSystemConfig
	if err := c.get(ctx, "/self/system", &sys); err == nil {
		cfg.System = &models.SystemConfig{
			StagingMode: sys.StagingMode,
			MinFanSpeed: sys.MinFanSpeed,
			SMPTE2022_7: sys.SMPTENetwork.ST20227.Class,
		}
	}

	var proto selfProtocols
	if err := c.get(ctx, "/self/protocols", &proto); err == nil {
		cfg.Protocols = &models.ProtocolsConfig{
			MDNSEnable:            proto.MDNSEnable,
			EmberServerPort:       proto.EmberServerPort,
			SAPAnnouncementEnable: proto.SAPAnnouncementEnable,
		}
	}

	var sl selfSyslog
	if err := c.get(ctx, "/self/syslog", &sl); err == nil {
		cfg.Syslog = &models.SyslogConfig{
			Server:     sl.Config.Server,
			Port:       sl.Config.Port,
			Enable:     sl.Config.Enable,
			Monitoring: sl.Monitoring,
		}
	}

	var routes map[string]struct {
		Destination string `json:"destination"`
		Gateway     string `json:"gateway"`
	}
	if err := c.get(ctx, "/self/static_route", &routes); err == nil {
		names := make([]string, 0, len(routes))
		for name := range routes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			r := routes[name]
			// Skip empty/unused slots (default route to 0.0.0.0).
			if r.Destination == "0.0.0.0/0" && r.Gateway == "0.0.0.0" {
				continue
			}
			cfg.StaticRoutes = append(cfg.StaticRoutes, models.StaticRoute{
				Name:        name,
				Destination: r.Destination,
				Gateway:     r.Gateway,
			})
		}
	}

	var dns selfDiagDNS
	if err := c.get(ctx, "/self/diag/dns", &dns); err == nil {
		cfg.DNS = &models.DNSConfig{
			ServerAddress: dns.DNS.ServerAddress,
			DomainName:    dns.DNS.DomainName,
		}
	}

	return cfg, nil
}

// --- Write methods (PUT). Each writes only the section it is given. ---

// UpdateNetwork writes /self/ipconfig. NOTE: the device reboots to apply.
func (c *EmsfpClient) UpdateNetwork(ctx context.Context, n models.NetworkUpdate) error {
	return c.put(ctx, "/self/ipconfig", n)
}

// UpdateProtocols writes /self/protocols (mDNS, Ember+, SAP).
func (c *EmsfpClient) UpdateProtocols(ctx context.Context, p models.ProtocolsConfig) error {
	return c.put(ctx, "/self/protocols", p)
}

// UpdateSyslog writes /self/syslog (server/port/enable + monitoring events).
func (c *EmsfpClient) UpdateSyslog(ctx context.Context, s models.SyslogUpdate) error {
	body := map[string]interface{}{
		"config": map[string]interface{}{
			"server": s.Server,
			"port":   s.Port,
			"enable": s.Enable,
		},
	}
	if s.Monitoring != nil {
		body["monitoring"] = s.Monitoring
	}
	return c.put(ctx, "/self/syslog", body)
}

// UpdateStaticRoutes writes /self/static_route. The device exposes a fixed set
// of route slots; unused slots are sent as the default 0.0.0.0/0 → 0.0.0.0.
func (c *EmsfpClient) UpdateStaticRoutes(ctx context.Context, routes []models.StaticRoute) error {
	const slots = 5
	body := make(map[string]map[string]string, slots)
	for i := 1; i <= slots; i++ {
		body[fmt.Sprintf("route_%d", i)] = map[string]string{
			"destination": "0.0.0.0/0",
			"gateway":     "0.0.0.0",
		}
	}
	for i, r := range routes {
		if i >= slots {
			break
		}
		body[fmt.Sprintf("route_%d", i+1)] = map[string]string{
			"destination": r.Destination,
			"gateway":     r.Gateway,
		}
	}
	return c.put(ctx, "/self/static_route", body)
}

// Reboot triggers a device reboot via /self/system.
func (c *EmsfpClient) Reboot(ctx context.Context) error {
	return c.put(ctx, "/self/system", map[string]string{"reboot": "1"})
}

// ConfigReset resets device configuration via /self/system. scope is one of
// flows, application, generic, system. The device reboots to apply.
func (c *EmsfpClient) ConfigReset(ctx context.Context, scope string) error {
	return c.put(ctx, "/self/system", map[string]string{"config_reset": scope})
}

// asString coerces a JSON value (string or number) to a string for display.
func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%g", t)
	default:
		return ""
	}
}

// CheckReachability performs a simple GET to /self/information and measures response time.
func (c *EmsfpClient) CheckReachability(ctx context.Context) (reachable bool, responseMs int64, err error) {
	start := time.Now()
	var info selfInformation
	err = c.get(ctx, "/self/information", &info)
	responseMs = time.Since(start).Milliseconds()
	reachable = err == nil
	return
}

// TCPProbe performs a lightweight L4 connectivity check by dialing the device's
// management port, independent of the REST API. This is the privilege-free
// alternative to an ICMP echo (raw ICMP sockets require elevated privileges on
// Windows; see ISSUES.md). Returns whether the port accepted a connection and
// how long the handshake took.
func ProbeTCP(ctx context.Context, ip, port string, timeout time.Duration) (reachable bool, responseMs int64) {
	if port == "" {
		port = "80"
	}
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
	responseMs = time.Since(start).Milliseconds()
	if err != nil {
		return false, responseMs
	}
	_ = conn.Close()
	return true, responseMs
}

// ProbeICMP performs an ICMP echo (ping) reachability check by invoking the
// operating system's `ping` command — privilege-free, unlike a raw ICMP socket.
// This is used for management paths that answer ICMP but not TCP (e.g. an EM6's
// second/Blue interface, which does not run the HTTP management server).
func ProbeICMP(ctx context.Context, ip string, timeout time.Duration) (reachable bool, responseMs int64) {
	cmd := pingCommand(ctx, ip, timeout)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Milliseconds()

	if !pingSucceeded(string(out), err == nil) {
		return false, elapsed
	}
	if rtt := parsePingMs(string(out)); rtt > 0 {
		return true, rtt
	}
	return true, elapsed
}

// pingCommand builds the platform-appropriate single-echo ping with a timeout.
func pingCommand(ctx context.Context, ip string, timeout time.Duration) *exec.Cmd {
	ms := int(timeout / time.Millisecond)
	if ms < 1 {
		ms = 1000
	}
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "ping", "-n", "1", "-w", strconv.Itoa(ms), ip)
	}
	// Linux/macOS: -c 1 one echo, -W timeout in (whole) seconds.
	secs := (ms + 999) / 1000
	if secs < 1 {
		secs = 1
	}
	return exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(secs), ip)
}

// pingSucceeded reports a real echo reply. It requires both a zero exit code and
// a "ttl=" marker so that Windows' "Destination host unreachable" replies (which
// can exit 0 yet carry no TTL) are correctly treated as unreachable.
func pingSucceeded(out string, exitOK bool) bool {
	return exitOK && strings.Contains(strings.ToLower(out), "ttl=")
}

var pingTimeRe = regexp.MustCompile(`(?i)time[=<]\s*([0-9.]+)`)

// parsePingMs extracts the round-trip time (ms) from ping output, rounded to the
// nearest millisecond. Returns 0 when no time field is present.
func parsePingMs(out string) int64 {
	m := pingTimeRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	if f < 1 {
		return 1
	}
	return int64(f + 0.5)
}
