import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { Device } from '../types/device';

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
    mutationFn: (device: Omit<Device, 'id' | 'created_at' | 'updated_at' | 'status'>) =>
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
