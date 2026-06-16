import { Link } from 'react-router-dom'
import { AlertTriangle, ShieldCheck, ChevronRight } from 'lucide-react'
import { clsx } from 'clsx'
import { useFleetAlarms } from '../hooks/useDevices'
import { formatRelativeTime } from '../utils/time'

// FleetAlarmPanel shows every active alarm across the fleet on the dashboard.
export function FleetAlarmPanel() {
  const { data, isLoading } = useFleetAlarms()
  const alarms = data?.alarms ?? []

  if (isLoading) {
    return <div className="card p-4 h-24 animate-pulse" />
  }

  if (alarms.length === 0) {
    return (
      <div className="card p-4 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-emerald-500/15 flex items-center justify-center shrink-0">
          <ShieldCheck className="w-5 h-5 text-emerald-400" />
        </div>
        <div>
          <p className="text-sm font-medium text-slate-200">All clear</p>
          <p className="text-xs text-slate-500">No active alarms across the fleet.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="card overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-surface-700">
        <div className="flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 text-amber-400" />
          <h3 className="text-sm font-semibold text-slate-200">Active Alarms</h3>
          <span className="text-xs px-1.5 py-0.5 rounded-full bg-amber-500/15 text-amber-400">{alarms.length}</span>
        </div>
      </div>
      <div className="divide-y divide-surface-800 max-h-72 overflow-y-auto">
        {alarms.map((a, i) => (
          <Link
            key={`${a.device_id}-${i}`}
            to={`/devices/${a.device_id}`}
            className="flex items-center gap-3 px-4 py-2.5 hover:bg-surface-800/50 transition-colors group"
          >
            <span className={clsx(
              'status-dot shrink-0',
              a.status === 'critical' ? 'status-critical' : a.status === 'offline' ? 'status-offline' : 'status-warning',
            )} />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium text-slate-300 truncate">{a.device_name || a.device_id}</p>
              <p className="text-xs text-slate-500 font-mono truncate">{a.message}</p>
            </div>
            {a.polled_at && (
              <span className="text-xs text-slate-600 shrink-0">{formatRelativeTime(a.polled_at)}</span>
            )}
            <ChevronRight className="w-3.5 h-3.5 text-slate-600 group-hover:text-slate-400 shrink-0" />
          </Link>
        ))}
      </div>
    </div>
  )
}
