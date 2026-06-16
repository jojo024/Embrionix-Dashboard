import { LayoutGrid, Table2, AlertTriangle, CheckCircle2, WifiOff, RefreshCw } from 'lucide-react';
import { useState } from 'react';
import { clsx } from 'clsx';
import { useDevices, useSummary } from '../hooks/useDevices';
import { DeviceCard } from '../components/DeviceCard';
import { DeviceTable } from '../components/DeviceTable';
import { StatusBadge } from '../components/StatusBadge';
import { FleetAlarmPanel } from '../components/FleetAlarmPanel';
import { RefreshCountdown } from '../components/RefreshCountdown';
import type { DeviceStatus } from '../types/device';

function SummaryCard({ label, value, status, icon: Icon }: {
  label: string;
  value: number;
  status?: DeviceStatus;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <div className="card p-4 flex items-center gap-4">
      <div className={clsx(
        'w-10 h-10 rounded-lg flex items-center justify-center shrink-0',
        status === 'online' && 'bg-emerald-500/15',
        status === 'offline' && 'bg-slate-700/50',
        status === 'warning' && 'bg-amber-500/15',
        status === 'critical' && 'bg-red-500/15',
        !status && 'bg-brand-500/15',
      )}>
        <Icon className={clsx(
          'w-5 h-5',
          status === 'online' && 'text-emerald-400',
          status === 'offline' && 'text-slate-400',
          status === 'warning' && 'text-amber-400',
          status === 'critical' && 'text-red-400',
          !status && 'text-brand-400',
        )} />
      </div>
      <div>
        <p className="text-2xl font-bold text-slate-100">{value}</p>
        <p className="text-xs text-slate-500">{label}</p>
      </div>
    </div>
  );
}

type ViewMode = 'card' | 'table';
type Filter = DeviceStatus | 'all';

export function Dashboard() {
  const [view, setView] = useState<ViewMode>('card');
  const [filter, setFilter] = useState<Filter>('all');
  const { data: deviceData, isLoading, refetch, isFetching, dataUpdatedAt } = useDevices();
  const { data: summary } = useSummary();

  const devices = deviceData?.devices ?? [];
  const filtered = filter === 'all' ? devices : devices.filter(d => d.status === filter);

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-100">Device Overview</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            {deviceData?.total ?? 0} devices registered
          </p>
        </div>
        <div className="flex items-center gap-3">
          <RefreshCountdown intervalSeconds={30} lastUpdated={dataUpdatedAt} isFetching={isFetching} />
          <button
            className="btn-secondary"
            onClick={() => refetch()}
            disabled={isFetching}
          >
            <RefreshCw className={clsx('w-4 h-4', isFetching && 'animate-spin')} />
            Refresh
          </button>
        </div>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <SummaryCard label="Online" value={summary.online} status="online" icon={CheckCircle2} />
          <SummaryCard label="Offline" value={summary.offline} status="offline" icon={WifiOff} />
          <SummaryCard label="Warning" value={summary.warning} status="warning" icon={AlertTriangle} />
          <SummaryCard label="Critical" value={summary.critical} status="critical" icon={AlertTriangle} />
        </div>
      )}

      {/* Fleet-wide alarm panel */}
      <FleetAlarmPanel />

      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3">
        {/* Status filter */}
        <div className="flex items-center gap-1.5 flex-wrap">
          {(['all', 'online', 'warning', 'critical', 'offline', 'unknown'] as const).map(f => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                filter === f
                  ? 'bg-surface-700 text-slate-100'
                  : 'text-slate-500 hover:text-slate-300 hover:bg-surface-800',
              )}
            >
              {f === 'all' ? `All (${devices.length})` : (
                <span className="flex items-center gap-1.5">
                  <StatusBadge status={f} showDot />
                  {summary?.[f] ?? 0}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* View toggle */}
        <div className="flex items-center gap-1 bg-surface-800 rounded-lg p-0.5">
          <button
            className={clsx('p-1.5 rounded-md transition-colors', view === 'card' ? 'bg-surface-700 text-slate-100' : 'text-slate-500 hover:text-slate-300')}
            onClick={() => setView('card')}
          >
            <LayoutGrid className="w-4 h-4" />
          </button>
          <button
            className={clsx('p-1.5 rounded-md transition-colors', view === 'table' ? 'bg-surface-700 text-slate-100' : 'text-slate-500 hover:text-slate-300')}
            onClick={() => setView('table')}
          >
            <Table2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="card p-4 h-48 animate-pulse">
              <div className="h-4 bg-surface-700 rounded w-3/4 mb-3" />
              <div className="h-3 bg-surface-800 rounded w-1/2 mb-6" />
              <div className="h-3 bg-surface-800 rounded w-full mb-2" />
              <div className="h-3 bg-surface-800 rounded w-4/5" />
            </div>
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="card p-12 text-center">
          <p className="text-slate-500">
            {devices.length === 0
              ? 'No devices added yet. Go to Settings → Devices to add your first EM6 device.'
              : `No devices match the "${filter}" filter.`}
          </p>
        </div>
      ) : view === 'card' ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filtered.map(device => (
            <DeviceCard key={device.id} device={device} />
          ))}
        </div>
      ) : (
        <DeviceTable devices={filtered} />
      )}
    </div>
  );
}
