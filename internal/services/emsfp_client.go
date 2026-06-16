package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/embrionix/dashboard/internal/models"
)

// EmsfpClient communicates with a single Embrionix EM6 device via its REST API.
// Base URL: http://<ip>/emsfp/node/v1
type EmsfpClient struct {
	httpClient *http.Client
	baseURL    string
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
				DisableKeepAlives: true,
			},
		},
	}
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

// Poll fetches all relevant data from the device and returns a DevicePollingData.
func (c *EmsfpClient) Poll(ctx context.Context) (*models.DevicePollingData, error) {
	data := &models.DevicePollingData{}

	// Fetch /self/information
	var info selfInformation
	if err := c.get(ctx, "/self/information", &info); err != nil {
		return nil, fmt.Errorf("self/information: %w", err)
	}
	data.CurrentVersion = info.CurrentVersion
	data.EmsfpVersion = info.EmsfpVersion
	data.DeviceType = info.Type
	data.PlatformHWVersion = info.PlatformHWVersion

	// Fetch /self/ipconfig
	var ipcfg selfIPConfig
	if err := c.get(ctx, "/self/ipconfig", &ipcfg); err == nil {
		data.Hostname = ipcfg.Hostname
		data.IPAddress = ipcfg.IPAddr
		data.DHCPEnable = ipcfg.DHCPEnable
		data.MACAddress = ipcfg.LocalMAC
	}

	// Fetch /self/system (temp, fan, uptime)
	var sys selfSystem
	if err := c.get(ctx, "/self/system", &sys); err == nil {
		data.CoreTemp = sys.CoreTemp
		data.FanSpeed = sys.FanSpeed
		data.CoreVoltage = sys.CoreVoltage
		data.Uptime = sys.Uptime
	}

	// Fetch /telemetry/node
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

	// Fetch /telemetry/ports
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

	// Gather alarms from SFP DDM data - attempt to get port list, then fetch each
	portListRaw := []string{}
	if err := c.get(ctx, "/port", &portListRaw); err == nil {
		for _, portEntry := range portListRaw {
			portID := portEntry
			// Remove trailing slash if present
			if len(portID) > 0 && portID[len(portID)-1] == '/' {
				portID = portID[:len(portID)-1]
			}
			var pi portInfo
			if err := c.get(ctx, "/port/"+portID, &pi); err == nil {
				pd := models.PortDetail{
					PortID:  portID,
					Link:    pi.Link,
					Speed:   pi.Speed,
					SFPType: pi.SFPType,
				}
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

					// Collect active alarms
					if ddm.AlarmStatus.HighTemperature {
						data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: High temperature alarm", portID))
					}
					if ddm.AlarmStatus.HighRxPower {
						data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: High RX power alarm", portID))
					}
					if ddm.AlarmStatus.LowRxPower {
						data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: Low RX power alarm", portID))
					}
					if ddm.AlarmStatus.LowTxPower {
						data.Alarms = append(data.Alarms, fmt.Sprintf("Port %s: Low TX power alarm", portID))
					}
				}
				data.PortDetails = append(data.PortDetails, pd)
			}
		}
	}

	return data, nil
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
