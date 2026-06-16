export type DeviceStatus = 'online' | 'offline' | 'warning' | 'critical' | 'unknown';

export interface DDMValue {
  high_alarm: number;
  low_alarm: number;
  high_warning: number;
  low_warning: number;
  current: number;
}

export interface DDMAlarmStatus {
  high_temperature: boolean;
  low_temperature: boolean;
  high_vcc: boolean;
  low_vcc: boolean;
  high_tx_bias: boolean;
  low_tx_bias: boolean;
  high_tx_power: boolean;
  low_tx_power: boolean;
  high_rx_power: boolean;
  low_rx_power: boolean;
}

export interface SFPDDM {
  temperature: DDMValue;
  vcc: DDMValue;
  tx_bias: DDMValue;
  tx_power: DDMValue;
  rx_power: DDMValue;
  alarm_status: DDMAlarmStatus;
  warning_status: DDMAlarmStatus;
}

export interface PortDetail {
  port_id: string;
  link: string;
  speed: string;
  sfp_type: string;
  ddm?: SFPDDM;
}

export interface PortTelemetry {
  port: number;
  temperature: number;
  tx_power: number;
  rx_power: number;
}

export interface PTPStatus {
  status_code: string;
  status_label: string;
  locked: boolean;
  master_ip: string;
  offset_from_master: number;
  mean_delay: number;
  sync_counter: number;
  delay_request_counter: number;
  coarse_unlock: boolean;
  unlock: boolean;
}

export interface FirmwareSlot {
  slot: number;
  product_id: number;
  desc: string;
  version: string;
  active: boolean;
  default: boolean;
}

export interface EthernetStats {
  tx_packets: string;
  rx_packets: string;
  rx_error: string;
  tx_rate: string;
  rx_rate: string;
}

export interface NetworkInterface {
  name: string;
  static_ip: string;
  static_gateway: string;
  current_ip: string;
  current_gateway: string;
  dhcp: boolean;
  vlan: number;
}

export interface LLDPNeighbor {
  chassis_id: string;
  port_id: string;
  ttl: string;
}

export interface MediaDeviceTelemetry {
  device: string;
  channel: number;
  type: string;
  valid: boolean;
  flow_count: number;
  total_pkts: number;
}

export interface DevicePollingData {
  // /self/information
  current_version: string;
  emsfp_version: string;
  device_type: string;
  platform_hw_version: string;

  // /self/system
  core_temp: number;
  fan_speed: number;
  core_voltage: number;
  uptime: string;

  // /self/ipconfig
  hostname: string;
  ip_addr: string;
  dhcp_enable: string;
  local_mac: string;

  // refclk
  refclk_status: string;
  grandmaster_id: string;
  offset_from_master: number;

  // /self/diag/refclk - detailed PTP
  ptp?: PTPStatus;

  // /self/firmware
  firmware_slots?: FirmwareSlot[];

  // /self/license
  licenses?: Record<string, string>;

  // /self/diag/ethernet
  ethernet?: EthernetStats;

  // /self/diag/common
  video_bandwidth_usage?: string;
  watchdog_status?: string;
  ipv4_packet_drop?: string;

  // /self/interfaces
  interfaces?: NetworkInterface[];

  // /lldp
  lldp?: LLDPNeighbor;

  // /telemetry/devices
  media_devices?: MediaDeviceTelemetry[];

  // /sdi
  sdi_bit_rate?: string;

  // SFP
  ports: PortTelemetry[];
  port_details: PortDetail[];
  alarms: string[];
}

export interface Device {
  id: string;
  name: string;
  description: string;
  location: string;
  rack: string;
  serial_number: string;
  model: string;
  firmware_version: string;
  management_ip_red: string;
  management_ip_blue: string;
  tags: string;
  notes: string;
  monitoring_enabled: boolean;
  created_at: string;
  updated_at: string;

  // Runtime (from polling)
  status: DeviceStatus;
  last_polled_at?: string;
  reachable_red?: boolean;
  reachable_blue?: boolean;
  polling_data?: DevicePollingData;
}

export interface DeviceListResponse {
  devices: Device[];
  total: number;
}

export interface PollResult {
  id: number;
  device_id: string;
  polled_at: string;
  reachable: boolean;
  response_ms: number;
  core_temp?: number;
  fan_speed?: number;
  core_voltage?: number;
  port0_tx_power?: number;
  port0_rx_power?: number;
  port0_temp?: number;
  port1_tx_power?: number;
  port1_rx_power?: number;
  port1_temp?: number;
  ptp_locked?: boolean;
  ptp_offset?: number;
  reachable_red?: boolean;
  reachable_blue?: boolean;
  error_message?: string;
}

export interface DashboardSummary {
  total: number;
  online: number;
  offline: number;
  warning: number;
  critical: number;
  unknown: number;
}

export interface FleetAlarm {
  device_id: string;
  device_name: string;
  status: DeviceStatus;
  message: string;
  polled_at?: string;
}

export interface FleetAlarmsResponse {
  alarms: FleetAlarm[];
  total: number;
}

export interface AlertEvent {
  id: number;
  device_id: string;
  device_name: string;
  from_status: DeviceStatus;
  to_status: DeviceStatus;
  message: string;
  created_at: string;
}

export interface AlertHistoryResponse {
  alerts: AlertEvent[];
  total: number;
}

export interface RuntimeConfig {
  polling: {
    interval_seconds: number;
    timeout_seconds: number;
    icmp_enabled: boolean;
    history_retention_days: number;
  };
  alerting: {
    temp_warning_c: number;
    temp_critical_c: number;
    response_warning_ms: number;
    webhook_enabled: boolean;
    webhook_on: string[];
  };
}
