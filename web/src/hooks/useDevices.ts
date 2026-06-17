import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { Device, NetworkUpdate, SyslogUpdate, ProtocolsConfig, StaticRoute, ConfigResetScope } from '../types/device';

export const DEVICES_KEY = ['devices'] as const;
export const SUMMARY_KEY = ['summary'] as const;

export function useDevices() {
  return useQuery({
    queryKey: DEVICES_KEY,
    queryFn: () => api.listDevices(),
    refetchInterval: 30_000,
  });
}

export function useDevice(id: string) {
  return useQuery({
    queryKey: ['device', id],
    queryFn: () => api.getDevice(id),
    enabled: !!id,
    refetchInterval: 30_000,
  });
}

export function useSummary() {
  return useQuery({
    queryKey: SUMMARY_KEY,
    queryFn: () => api.getSummary(),
    refetchInterval: 30_000,
  });
}

export const ALARMS_KEY = ['alarms'] as const;

export function useFleetAlarms() {
  return useQuery({
    queryKey: ALARMS_KEY,
    queryFn: () => api.getFleetAlarms(),
    refetchInterval: 30_000,
  });
}

export function useAlertHistory(deviceId?: string, limit = 100) {
  return useQuery({
    queryKey: ['alert-history', deviceId ?? 'all', limit],
    queryFn: () => api.getAlertHistory(deviceId, limit),
    refetchInterval: 30_000,
  });
}

// Read-only device configuration, fetched on demand (config changes rarely, so
// no background refetch). `enabled` defers the fetch until the tab is opened.
export function useDeviceConfig(id: string, enabled: boolean) {
  return useQuery({
    queryKey: ['device-config', id],
    queryFn: () => api.getDeviceConfig(id),
    enabled: enabled && !!id,
    staleTime: 60_000,
  });
}

export function useAuditLog(deviceId?: string, limit = 100) {
  return useQuery({
    queryKey: ['audit-log', deviceId ?? 'all', limit],
    queryFn: () => api.getAuditLog(deviceId, limit),
    refetchInterval: 30_000,
  });
}

// --- Config-write mutations. All invalidate the device-config and
// audit-log queries so the UI reflects the new state after a write. ---
function useConfigMutation<T>(id: string, fn: (vars: T) => Promise<unknown>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['device-config', id] });
      qc.invalidateQueries({ queryKey: ['audit-log'] });
    },
  });
}

export function useUpdateNetwork(id: string) {
  return useConfigMutation<NetworkUpdate>(id, (body) => api.updateNetwork(id, body));
}
export function useUpdateProtocols(id: string) {
  return useConfigMutation<ProtocolsConfig>(id, (body) => api.updateProtocols(id, body));
}
export function useUpdateSyslog(id: string) {
  return useConfigMutation<SyslogUpdate>(id, (body) => api.updateSyslog(id, body));
}
export function useUpdateRoutes(id: string) {
  return useConfigMutation<StaticRoute[]>(id, (routes) => api.updateRoutes(id, routes));
}
export function useRebootDevice(id: string) {
  return useConfigMutation<void>(id, () => api.rebootDevice(id));
}
export function useConfigReset(id: string) {
  return useConfigMutation<ConfigResetScope>(id, (scope) => api.configReset(id, scope));
}

export function useDeviceHistory(id: string) {
  return useQuery({
    queryKey: ['device-history', id],
    queryFn: () => api.getDeviceHistory(id, 200),
    enabled: !!id,
    refetchInterval: 60_000,
  });
}

// Small recent-history slice for device-card sparklines.
export function useDeviceSparkline(id: string) {
  return useQuery({
    queryKey: ['device-sparkline', id],
    queryFn: () => api.getDeviceHistory(id, 24),
    enabled: !!id,
    refetchInterval: 60_000,
  });
}

export function useCreateDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (device: Omit<Device, 'id' | 'created_at' | 'updated_at' | 'status' | 'last_polled_at' | 'reachable_red' | 'reachable_blue' | 'polling_data' | 'slow_response_count'>) =>
      api.createDevice(device),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: DEVICES_KEY });
      qc.invalidateQueries({ queryKey: SUMMARY_KEY });
    },
  });
}

export function useUpdateDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (device: Device) => api.updateDevice(device),
    onSuccess: () => qc.invalidateQueries({ queryKey: DEVICES_KEY }),
  });
}

export function useDeleteDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteDevice(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: DEVICES_KEY });
      qc.invalidateQueries({ queryKey: SUMMARY_KEY });
    },
  });
}

export function usePollDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.pollDeviceNow(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: DEVICES_KEY }),
  });
}
