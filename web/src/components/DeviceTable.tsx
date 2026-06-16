import { useNavigate } from 'react-router-dom';
import { clsx } from 'clsx';
import type { Device } from '../types/device';
import { StatusBadge } from './StatusBadge';
import { formatRelativeTime } from '../utils/time';

interface Props {
  devices: Device[];
}

export function DeviceTable({ devices }: Props) {
  const navigate = useNavigate();

  return (
    <div className="card overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-surface-700">
              {['Status', 'Name', 'Model', 'Location', 'IP (Red)', 'IP (Blue)', 'Temp', 'Last Poll'].map(h => (
                <th key={h} className="px-4 py-3 text-left text-xs font-medium text-slate-500 whitespace-nowrap">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-800">
            {devices.map(device => (
              <tr
                key={device.id}
                className="hover:bg-surface-800 cursor-pointer transition-colors"
                onClick={() => navigate(`/devices/${device.id}`)}
              >
                <td className="px-4 py-3">
                  <StatusBadge status={device.status} />
                </td>
                <td className="px-4 py-3">
                  <div className="font-medium text-slate-100">{device.name}</div>
                  {device.polling_data?.hostname && (
                    <div className="text-xs text-slate-500 font-mono">{device.polling_data.hostname}</div>
                  )}
                </td>
                <td className="px-4 py-3 text-slate-400 font-mono text-xs">
                  {device.model || device.polling_data?.device_type || '—'}
                </td>
                <td className="px-4 py-3 text-slate-400">
                  {[device.location, device.rack].filter(Boolean).join(' / ') || '—'}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    {device.reachable_red !== undefined && (
                      <span className={clsx('status-dot', device.reachable_red ? 'status-online' : 'status-offline')} />
                    )}
                    <span className="font-mono text-xs text-slate-400">{device.management_ip_red || '—'}</span>
                  </div>
                </td>
                <td className="px-4 py-3">
                  <span className="font-mono text-xs text-slate-400">{device.management_ip_blue || '—'}</span>
                </td>
                <td className="px-4 py-3 font-mono text-xs">
                  {device.polling_data?.core_temp
                    ? <span className={device.polling_data.core_temp > 70 ? 'text-amber-400' : 'text-slate-400'}>
                        {device.polling_data.core_temp.toFixed(1)}°C
                      </span>
                    : <span className="text-slate-600">—</span>}
                </td>
                <td className="px-4 py-3 text-xs text-slate-500">
                  {device.last_polled_at ? formatRelativeTime(device.last_polled_at) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
