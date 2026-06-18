import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Wifi, WifiOff, Thermometer, Wind, Activity, AlertTriangle, Zap, X } from 'lucide-react';
import { clsx } from 'clsx';
import type { Device } from '../types/device';
import { StatusBadge } from './StatusBadge';
import { Sparkline } from './Sparkline';
import { useDeviceSparkline } from '../hooks/useDevices';
import { formatRelativeTime } from '../utils/time';
import { readableFirmware } from '../utils/firmware';
import { ipConfigIssues } from '../utils/ipconfig';
import { powerTodBm, txPowerClass } from '../utils/optics';

interface Props {
  device: Device;
}

function MetricChip({ icon: Icon, value, label, warn }: {
  icon: React.ComponentType<{ className?: string }>;
  value: string;
  label: string;
  warn?: boolean;
}) {
  return (
    <div className={clsx(
      'flex items-center gap-1.5 px-2 py-1 rounded-md text-xs',
      warn ? 'bg-amber-500/10 text-amber-400' : 'bg-surface-800 text-slate-400',
    )}>
      <Icon className="w-3 h-3 shrink-0" />
      <span className="font-mono">{value}</span>
      <span className="text-slate-500">{label}</span>
    </div>
  );
}

function NetworkStatus({ label, reachable }: { label: string; reachable?: boolean }) {
  if (reachable === undefined) {
    return (
      <div className="flex items-center gap-1 text-xs text-slate-500">
        <span className="w-1.5 h-1.5 rounded-full bg-slate-600" />
        <span>{label}</span>
      </div>
    );
  }
  return (
    <div className={clsx('flex items-center gap-1 text-xs', reachable ? 'text-emerald-400' : 'text-slate-500')}>
      {reachable
        ? <Wifi className="w-3 h-3" />
        : <WifiOff className="w-3 h-3" />}
      <span>{label}</span>
    </div>
  );
}

export function DeviceCard({ device }: Props) {
  const navigate = useNavigate();
  const pd = device.polling_data;
  const ipIssues = ipConfigIssues(device);
  const { data: spark } = useDeviceSparkline(device.id);
  const [dismissedAlarms, setDismissedAlarms] = useState<boolean>(false);
  const [showTempGraph, setShowTempGraph] = useState<boolean>(false);

  const tempSeries = (spark ?? [])
    .slice()
    .reverse()
    .map(r => r.core_temp)
    .filter((v): v is number => v != null);

  // Filter alarms: if dismissed, don't show
  const visibleAlarms = dismissedAlarms ? [] : (pd?.alarms ?? []);

  const borderColor = {
    online: 'border-emerald-500/20 hover:border-emerald-500/40',
    warning: 'border-amber-500/30 hover:border-amber-500/50',
    critical: 'border-red-500/30 hover:border-red-500/50',
    offline: 'border-surface-700 hover:border-surface-600',
    unknown: 'border-surface-700 hover:border-surface-600',
  }[device.status] ?? 'border-surface-700 hover:border-surface-600';

  const hasAlarms = visibleAlarms.length > 0;

  return (
    <div
      className={clsx(
        'card cursor-pointer transition-all duration-200 hover:bg-surface-800 hover:-translate-y-0.5',
        'hover:shadow-lg hover:shadow-black/30 group',
        borderColor,
      )}
      onClick={() => navigate(`/devices/${device.id}`)}
    >
      {/* Header */}
      <div className="flex items-start justify-between p-4 pb-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={clsx('status-dot', `status-${device.status}`)} />
            <h3 className="font-semibold text-slate-100 truncate text-sm">{device.name}</h3>
          </div>
          {device.model && (
            <p className="text-xs text-slate-500 mt-0.5 ml-4 font-mono">{device.model}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-2">
          {hasAlarms && (
            <button
              onClick={(e) => { e.stopPropagation(); setDismissedAlarms(true) }}
              className="group hover:bg-amber-500/20 p-1 rounded-md transition-colors"
              title="Dismiss alarms"
            >
              <AlertTriangle className="w-3.5 h-3.5 text-amber-400 group-hover:text-amber-300" />
            </button>
          )}
          {device.slow_response_count >= 3 && (
            <div className="flex items-center gap-1 px-2 py-0.5 rounded-md bg-orange-500/15 text-orange-400 text-xs font-medium">
              <Zap className="w-3 h-3" />
              slow
            </div>
          )}
          <StatusBadge status={device.status} />
        </div>
      </div>

      {/* Body */}
      <div className="px-4 pb-3 space-y-3">
        {/* Location */}
        {(device.location || device.rack) && (
          <p className="text-xs text-slate-500">
            {[device.location, device.rack].filter(Boolean).join(' · ')}
          </p>
        )}

        {/* Metrics row */}
        {pd && (
          <div className="flex flex-wrap items-center gap-1.5">
            {pd.core_temp > 0 && (
              <button
                onClick={(e) => { e.stopPropagation(); setShowTempGraph(!showTempGraph) }}
                className="hover:opacity-75 transition-opacity cursor-pointer"
                title="Click to toggle temperature graph"
              >
                <MetricChip
                  icon={Thermometer}
                  value={`${pd.core_temp.toFixed(1)}°C`}
                  label="temp"
                  warn={pd.core_temp > 70}
                />
              </button>
            )}
            {showTempGraph && tempSeries.length > 1 && (
              <Sparkline
                data={tempSeries}
                className="opacity-80"
                stroke={pd.core_temp > 70 ? '#f59e0b' : '#38bdf8'}
              />
            )}
            {pd.fan_speed > 0 && (
              <MetricChip icon={Wind} value={`${pd.fan_speed}`} label="rpm" />
            )}
            {pd.uptime && (
              <MetricChip icon={Activity} value={pd.uptime.split(',')[0]} label="" />
            )}
          </div>
        )}

        {/* Network status */}
        <div className="flex items-center gap-4">
          <NetworkStatus
            label={device.management_ip_red || 'Red—'}
            reachable={device.reachable_red}
          />
          {device.management_ip_blue && (
            <NetworkStatus
              label={device.management_ip_blue || 'Blue—'}
              reachable={device.reachable_blue}
            />
          )}
        </div>

        {/* IP config mismatch indicator */}
        {ipIssues.length > 0 && (
          <div
            className="flex items-center gap-1.5 text-xs text-amber-400"
            title={ipIssues.join('\n')}
          >
            <AlertTriangle className="w-3 h-3 shrink-0" />
            <span>IP config mismatch</span>
          </div>
        )}

        {/* SFP light levels — ports 3 & 5 only, with per-port LLDP neighbour.
            LLDP reports local interfaces as 1 & 2, which map to ports 3 & 5. */}
        {pd?.ports && (
          (() => {
            const relevantPorts = pd.ports.filter(p => p.port === 3 || p.port === 5)
            const neighbors = pd.lldp_neighbors ?? (pd.lldp ? [pd.lldp] : [])
            const portToInterface: Record<number, number> = { 3: 1, 5: 2 }
            const neighborFor = (port: number) => neighbors.find(n => n.interface === portToInterface[port])
            return relevantPorts.length > 0 ? (
              <div className="grid grid-cols-2 gap-1.5">
                {relevantPorts.map((p) => {
                  const n = neighborFor(p.port)
                  return (
                    <div key={p.port} className="bg-surface-800 rounded-md px-2 py-1">
                      <div className="flex items-center justify-between gap-2 mb-0.5">
                        <span className="text-xs text-slate-500 shrink-0">Port {p.port}</span>
                        {n?.port_id && (
                          <span
                            className="text-xs font-mono text-slate-400 truncate"
                            title={`LLDP neighbour: chassis ${n.chassis_id}, port ${n.port_id}`}
                          >
                            {n.port_id}
                          </span>
                        )}
                      </div>
                      <div className="space-y-0.5 text-xs font-mono">
                        <div className={txPowerClass(p.tx_power)}>TX {powerTodBm(p.tx_power)}</div>
                        <div className="text-green-400">RX {powerTodBm(p.rx_power)}</div>
                      </div>
                    </div>
                  )
                })}
              </div>
            ) : null
          })()
        )}

        {/* Dismiss alarms button — shows when alarms exist and not yet dismissed */}
        {(pd?.alarms && pd.alarms.length > 0 && !dismissedAlarms) && (
          <div className="flex items-center justify-between bg-amber-500/10 border border-amber-500/20 rounded-md px-2 py-1">
            <span className="text-xs text-amber-300 font-medium">{pd.alarms.length} alarm{pd.alarms.length !== 1 ? 's' : ''}</span>
            <button
              onClick={(e) => { e.stopPropagation(); setDismissedAlarms(true) }}
              className="p-0.5 hover:bg-amber-500/20 rounded transition-colors"
              title="Dismiss alarms"
            >
              <X className="w-3 h-3 text-amber-400" />
            </button>
          </div>
        )}
        {dismissedAlarms && (pd?.alarms?.length ?? 0) > 0 && (
          <div className="text-xs text-slate-500 px-2 py-1 border border-dashed border-slate-700 rounded-md">
            {(pd?.alarms?.length ?? 0)} alarm{(pd?.alarms?.length ?? 0) !== 1 ? 's' : ''} dismissed
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-surface-800 flex items-center justify-between">
        <span className="text-xs text-slate-600 font-mono" title={pd?.current_version || device.firmware_version}>
          {readableFirmware(pd, device.firmware_version)}
        </span>
        <span className="text-xs text-slate-600">
          {device.last_polled_at ? formatRelativeTime(device.last_polled_at) : 'Never polled'}
        </span>
      </div>
    </div>
  );
}
