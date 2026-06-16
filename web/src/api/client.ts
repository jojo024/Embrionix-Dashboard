import type { Device, DeviceListResponse, DashboardSummary, PollResult, FleetAlarmsResponse, AlertHistoryResponse, RuntimeConfig, DeviceConfig, NetworkUpdate, SyslogUpdate, ProtocolsConfig, StaticRoute, ConfigResetScope, AuditLogResponse, AuditEvent } from '../types/device';

const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8081';

export const API_BASE = BASE;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as unknown as T;
  return res.json();
}

export const api = {
  // ---- Devices ----
  listDevices: (): Promise<DeviceListResponse> =>
    request('/api/v1/devices'),

  getDevice: (id: string): Promise<Device> =>
    request(`/api/v1/devices/${id}`),

  createDevice: (device: Omit<Device, 'id' | 'created_at' | 'updated_at' | 'status'>): Promise<Device> =>
    request('/api/v1/devices', { method: 'POST', body: JSON.stringify(device) }),

  updateDevice: (device: Device): Promise<Device> =>
    request(`/api/v1/devices/${device.id}`, { method: 'PUT', body: JSON.stringify(device) }),

  deleteDevice: (id: string): Promise<void> =>
    request(`/api/v1/devices/${id}`, { method: 'DELETE' }),

  // ---- Monitoring ----
  getSummary: (): Promise<DashboardSummary> =>
    request('/api/v1/summary'),

  getFleetAlarms: (): Promise<FleetAlarmsResponse> =>
    request('/api/v1/alarms'),

  getAlertHistory: (deviceId?: string, limit = 100): Promise<AlertHistoryResponse> =>
    request(`/api/v1/alerts?limit=${limit}${deviceId ? `&device=${deviceId}` : ''}`),

  historyCsvUrl: (deviceId: string): string =>
    `${BASE}/api/v1/devices/${deviceId}/history.csv`,

  getDeviceHistory: (id: string, limit = 100): Promise<PollResult[]> =>
    request(`/api/v1/devices/${id}/history?limit=${limit}`),

  pollDeviceNow: (id: string): Promise<{ reachable: boolean; polling_data?: Device['polling_data'] }> =>
    request(`/api/v1/devices/${id}/poll`, { method: 'POST' }),

  getDeviceConfig: (id: string): Promise<DeviceConfig> =>
    request(`/api/v1/devices/${id}/config`),

  // ---- Configuration writes + device actions (Phase 4b) ----
  updateNetwork: (id: string, body: NetworkUpdate): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/config/network`, { method: 'PUT', body: JSON.stringify(body) }),

  updateProtocols: (id: string, body: ProtocolsConfig): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/config/protocols`, { method: 'PUT', body: JSON.stringify(body) }),

  updateSyslog: (id: string, body: SyslogUpdate): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/config/syslog`, { method: 'PUT', body: JSON.stringify(body) }),

  updateRoutes: (id: string, routes: StaticRoute[]): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/config/routes`, { method: 'PUT', body: JSON.stringify({ routes }) }),

  rebootDevice: (id: string): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/reboot`, { method: 'POST' }),

  configReset: (id: string, scope: ConfigResetScope): Promise<{ ok: boolean; audit: AuditEvent }> =>
    request(`/api/v1/devices/${id}/config-reset`, { method: 'POST', body: JSON.stringify({ scope }) }),

  getAuditLog: (deviceId?: string, limit = 100): Promise<AuditLogResponse> =>
    request(`/api/v1/audit?limit=${limit}${deviceId ? `&device=${deviceId}` : ''}`),

  checkReachability: (id: string): Promise<Record<string, { ip: string; reachable: boolean; response_ms: number }>> =>
    request(`/api/v1/devices/${id}/reachability`),

  // ---- Settings ----
  getConfig: (): Promise<RuntimeConfig> =>
    request('/api/v1/config'),

  getSetting: (key: string): Promise<{ key: string; value: string }> =>
    request(`/api/v1/settings/${key}`),

  setSetting: (key: string, value: string): Promise<{ key: string; value: string }> =>
    request(`/api/v1/settings/${key}`, { method: 'PUT', body: JSON.stringify({ value }) }),

  // ---- Health ----
  health: (): Promise<{ status: string; uptime: string }> =>
    request('/health'),
};
