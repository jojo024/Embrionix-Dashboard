import { useState } from 'react'
import { RefreshCw, Wifi, WifiOff, AlertTriangle, CheckCircle2, Clock } from 'lucide-react'
import { clsx } from 'clsx'
import { useDevices, useSummary } from '../hooks/useDevices'
import { StatusBadge } from '../components/StatusBadge'
import { formatRelativeTime } from '../utils/time'
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Cell
} from 'recharts'

export function MonitoringPage() {
  const { data: deviceData, isLoading, refetch, isFetching } = useDevices()
  const { data: summary } = useSummary()
  const [sortBy, setSortBy] = useState<'name' | 'status' | 'temp'>('status')

  const devices = deviceData?.devices ?? []

  // Sort
  const sorted = [...devices].sort((a, b) => {
    if (sortBy === 'status') {
      const order: Record<string, number> = { critical: 0, warning: 1, offline: 2, online: 3, unknown: 4 }
      return (order[a.status] ?? 5) - (order[b.status] ?? 5)
    }
    if (sortBy === 'temp') {
      return (b.polling_data?.core_temp ?? 0) - (a.polling_data?.core_temp ?? 0)
    }
    return a.name.localeCompare(b.name)
  })

  // Status distribution chart data
  const chartData = summary
    ? [
        { name: 'Online', value: summary.online, color: '#34d399' },
        { name: 'Warning', value: summary.warning, color: '#fbbf24' },
        { name: 'Critical', value: summary.critical, color: '#f87171' },
        { name: 'Offline', value: summary.offline, color: '#475569' },
        { name: 'Unknown', value: summary.unknown, color: '#334155' },
      ]
    : []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-100">Monitoring</h1>
          <p className="text-sm text-slate-500 mt-0.5">Real-time device health overview</p>
        </div>
        <button className="btn-secondary" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={clsx('w-4 h-4', isFetching && 'animate-spin')} />
          Refresh
        </button>
      </div>

      {/* Summary + chart */}
      {summary && (
        <div className="grid sm:grid-cols-3 gap-4">
          {/* Status distribution bar chart */}
          <div className="sm:col-span-2 card p-4">
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Status Distribution</h3>
            <ResponsiveContainer width="100%" height={120}>
              <BarChart data={chartData} layout="vertical" margin={{ left: 10 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" horizontal={false} />
                <XAxis type="number" tick={{ fill: '#64748b', fontSize: 10 }} />
                <YAxis type="category" dataKey="name" tick={{ fill: '#94a3b8', fontSize: 11 }} width={55} />
                <Tooltip
                  contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 8 }}
                  cursor={{ fill: '#ffffff08' }}
                />
                <Bar dataKey="value" radius={4} maxBarSize={20}>
                  {chartData.map((entry, index) => (
                    <Cell key={index} fill={entry.color} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Quick stats */}
          <div className="card p-4 space-y-3">
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Quick Stats</h3>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-emerald-400">
                <CheckCircle2 className="w-4 h-4" /> Healthy
              </div>
              <span className="text-lg font-bold text-slate-100">{summary.online}</span>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-amber-400">
                <AlertTriangle className="w-4 h-4" /> Needs attention
              </div>
              <span className="text-lg font-bold text-slate-100">{summary.warning + summary.critical}</span>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-slate-500">
                <WifiOff className="w-4 h-4" /> Unreachable
              </div>
              <span className="text-lg font-bold text-slate-100">{summary.offline}</span>
            </div>
            <div className="border-t border-surface-700 pt-3 flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Wifi className="w-4 h-4" /> Total
              </div>
              <span className="text-lg font-bold text-slate-100">{summary.total}</span>
            </div>
          </div>
        </div>
      )}

      {/* Sort controls */}
      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">Sort by:</span>
        {(['status', 'name', 'temp'] as const).map(s => (
          <button
            key={s}
            onClick={() => setSortBy(s)}
            className={clsx(
              'px-3 py-1 rounded-lg text-xs font-medium transition-colors',
              sortBy === s
                ? 'bg-surface-700 text-slate-100'
                : 'text-slate-500 hover:text-slate-300 hover:bg-surface-800',
            )}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {/* Device list */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="card h-16 animate-pulse" />
          ))}
        </div>
      ) : sorted.length === 0 ? (
        <div className="card p-12 text-center text-slate-500">
          No devices to monitor. Add devices in Settings → Devices.
        </div>
      ) : (
        <div className="card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-surface-700">
                  {['Status', 'Name', 'IP Red / Blue', 'Temp', 'Fan', 'SFP TX', 'SFP RX', 'PTP', 'Last Poll'].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-medium text-slate-500 whitespace-nowrap">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800">
                {sorted.map(device => {
                  const pd = device.polling_data
                  const primaryPort = pd?.ports?.[0]
                  return (
                    <tr key={device.id} className="hover:bg-surface-800/50 transition-colors">
                      <td className="px-4 py-3">
                        <StatusBadge status={device.status} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="font-medium text-slate-100">{device.name}</div>
                        {pd?.hostname && <div className="text-xs text-slate-500 font-mono">{pd.hostname}</div>}
                      </td>
                      <td className="px-4 py-3">
                        <div className="font-mono text-xs text-slate-400">
                          {device.management_ip_red || '—'}
                        </div>
                        {device.management_ip_blue && (
                          <div className="font-mono text-xs text-slate-600">
                            {device.management_ip_blue}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs">
                        {pd?.core_temp
                          ? <span className={pd.core_temp > 70 ? 'text-amber-400' : 'text-slate-300'}>
                              {pd.core_temp.toFixed(1)}°C
                            </span>
                          : <span className="text-slate-600">—</span>}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-slate-400">
                        {pd?.fan_speed ?? '—'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-blue-400">
                        {primaryPort?.tx_power ?? '—'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-green-400">
                        {primaryPort?.rx_power ?? '—'}
                      </td>
                      <td className="px-4 py-3 text-xs">
                        {pd?.refclk_status
                          ? <span className={pd.refclk_status === 'locked' ? 'text-emerald-400' : 'text-amber-400'}>
                              {pd.refclk_status}
                            </span>
                          : <span className="text-slate-600">—</span>}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5 text-xs text-slate-500">
                          <Clock className="w-3 h-3" />
                          {device.last_polled_at ? formatRelativeTime(device.last_polled_at) : '—'}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
