import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Server, Clock, Bell, Download, Info, ChevronRight, Layers } from 'lucide-react'
import { clsx } from 'clsx'
import { useQuery } from '@tanstack/react-query'
import { DevicesPage } from './DevicesPage'
import { useDevices } from '../hooks/useDevices'
import { useToast } from '../components/Toast'
import { api } from '../api/client'

type Tab = 'devices' | 'polling' | 'alerting' | 'bulk' | 'backup' | 'about'

const TABS: { id: Tab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'devices', label: 'Device Management', icon: Server },
  { id: 'polling', label: 'Polling Configuration', icon: Clock },
  { id: 'alerting', label: 'Alerting', icon: Bell },
  { id: 'bulk', label: 'Bulk Configuration', icon: Layers },
  { id: 'backup', label: 'Backup & Restore', icon: Download },
  { id: 'about', label: 'About', icon: Info },
]

function BulkConfigSettings() {
  const { data } = useDevices()
  const { notify } = useToast()
  const devices = data?.devices ?? []
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [section, setSection] = useState<'protocols' | 'syslog'>('syslog')
  const [syslog, setSyslog] = useState({ server: '', port: '514', enable: true })
  const [protocols, setProtocols] = useState({ mdns_enable: '1', ember_server_port: '3344', sap_announcement_enable: '0' })
  const [busy, setBusy] = useState(false)

  const toggle = (id: string) => setSelected(s => {
    const n = new Set(s)
    n.has(id) ? n.delete(id) : n.add(id)
    return n
  })

  const apply = async () => {
    if (selected.size === 0) { notify('error', 'Select at least one device.'); return }
    setBusy(true)
    try {
      const res = await api.bulkConfig(
        section === 'syslog'
          ? { device_ids: [...selected], section, syslog: { server: syslog.server, port: Number(syslog.port) || 514, enable: syslog.enable } }
          : { device_ids: [...selected], section, protocols },
      )
      const ok = res.results.filter(r => r.success).length
      const failed = res.results.length - ok
      notify(failed ? 'error' : 'success', `Applied to ${ok}/${res.results.length} device(s)${failed ? `; ${failed} failed` : ''}.`)
    } catch (e) {
      notify('error', `Bulk apply failed: ${(e as Error).message}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-lg space-y-5">
      <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg px-4 py-2.5 text-xs text-amber-300">
        Bulk changes are written to every selected live device and audited per device.
      </div>

      <div>
        <label className="label">Section</label>
        <div className="flex gap-1.5">
          {(['syslog', 'protocols'] as const).map(s => (
            <button key={s} onClick={() => setSection(s)}
              className={clsx('px-3 py-1.5 rounded-lg text-xs font-medium capitalize',
                section === s ? 'bg-surface-700 text-slate-100' : 'text-slate-500 hover:bg-surface-800')}>
              {s}
            </button>
          ))}
        </div>
      </div>

      {section === 'syslog' ? (
        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2"><label className="label">Syslog Server</label>
            <input className="input" value={syslog.server} placeholder="192.168.1.10" onChange={e => setSyslog({ ...syslog, server: e.target.value })} /></div>
          <div><label className="label">Port</label>
            <input className="input" value={syslog.port} onChange={e => setSyslog({ ...syslog, port: e.target.value })} /></div>
          <label className="flex items-center gap-2 text-xs text-slate-400 mt-6">
            <input type="checkbox" checked={syslog.enable} onChange={e => setSyslog({ ...syslog, enable: e.target.checked })} /> Enabled
          </label>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          <label className="flex items-center gap-2 text-xs text-slate-400">
            <input type="checkbox" checked={protocols.mdns_enable === '1'} onChange={e => setProtocols({ ...protocols, mdns_enable: e.target.checked ? '1' : '0' })} /> mDNS
          </label>
          <label className="flex items-center gap-2 text-xs text-slate-400">
            <input type="checkbox" checked={protocols.sap_announcement_enable === '1'} onChange={e => setProtocols({ ...protocols, sap_announcement_enable: e.target.checked ? '1' : '0' })} /> SAP Announce
          </label>
          <div className="col-span-2"><label className="label">Ember+ Port</label>
            <input className="input" value={protocols.ember_server_port} onChange={e => setProtocols({ ...protocols, ember_server_port: e.target.value })} /></div>
        </div>
      )}

      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="label mb-0">Target Devices ({selected.size})</label>
          <button className="text-xs text-brand-400 hover:text-brand-300"
            onClick={() => setSelected(selected.size === devices.length ? new Set() : new Set(devices.map(d => d.id)))}>
            {selected.size === devices.length ? 'Clear all' : 'Select all'}
          </button>
        </div>
        <div className="card divide-y divide-surface-800 max-h-60 overflow-y-auto">
          {devices.map(d => (
            <label key={d.id} className="flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-surface-800/50">
              <input type="checkbox" checked={selected.has(d.id)} onChange={() => toggle(d.id)} />
              <span className="text-xs text-slate-300">{d.name}</span>
              <span className="text-xs text-slate-600 font-mono ml-auto">{d.management_ip_red || d.management_ip_blue}</span>
            </label>
          ))}
          {devices.length === 0 && <p className="text-xs text-slate-500 p-3">No devices.</p>}
        </div>
      </div>

      <button className="btn-primary" onClick={apply} disabled={busy}>
        {busy ? 'Applying…' : `Apply to ${selected.size} device(s)`}
      </button>
    </div>
  )
}

function AlertingSettings() {
  const { data: config, isLoading } = useQuery({ queryKey: ['config'], queryFn: () => api.getConfig() })

  if (isLoading) return <div className="text-sm text-slate-500">Loading…</div>
  if (!config) return <div className="text-sm text-slate-500">Configuration unavailable.</div>

  const a = config.alerting
  const rows: [string, string][] = [
    ['Temperature warning', `≥ ${a.temp_warning_c} °C`],
    ['Temperature critical', `≥ ${a.temp_critical_c} °C`],
    ['Slow-response warning', `≥ ${a.response_warning_ms} ms`],
    ['Webhook notifications', a.webhook_enabled ? 'Enabled' : 'Disabled'],
    ['Notify on transition to', a.webhook_on.join(', ') || '—'],
  ]

  return (
    <div className="max-w-md space-y-4">
      <div className="card divide-y divide-surface-800">
        {rows.map(([label, value]) => (
          <div key={label} className="flex items-center justify-between px-4 py-3">
            <span className="text-xs text-slate-500">{label}</span>
            <span className="text-xs font-mono text-slate-300">{value}</span>
          </div>
        ))}
      </div>
      <p className="text-xs text-slate-500">
        Alerting thresholds and the notification webhook are set in
        {' '}<span className="font-mono text-slate-400">configs/config.yaml</span>{' '}
        (or <span className="font-mono text-slate-400">EMB_ALERTING_*</span> environment
        variables) and applied on startup. Status transitions are recorded in each
        device's Logs tab; configured destinations also receive a webhook.
      </p>
    </div>
  )
}

function PollingSettings() {
  const [interval, setIntervalVal] = useState('30')
  const [timeout, setTimeoutVal] = useState('10')
  const [retries, setRetries] = useState('2')
  const [saved, setSaved] = useState(false)

  const save = async () => {
    await Promise.all([
      api.setSetting('polling.interval_seconds', interval),
      api.setSetting('polling.timeout_seconds', timeout),
      api.setSetting('polling.retry_count', retries),
    ])
    setSaved(true)
    window.setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="max-w-md space-y-5">
      <div>
        <label className="label">Poll Interval (seconds)</label>
        <input type="number" min={10} max={3600} value={interval}
          onChange={e => setIntervalVal(e.target.value)} className="input" />
        <p className="text-xs text-slate-500 mt-1">How often to poll each device. Minimum 10s.</p>
      </div>
      <div>
        <label className="label">Request Timeout (seconds)</label>
        <input type="number" min={3} max={60} value={timeout}
          onChange={e => setTimeoutVal(e.target.value)} className="input" />
      </div>
      <div>
        <label className="label">Retry Count</label>
        <input type="number" min={0} max={5} value={retries}
          onChange={e => setRetries(e.target.value)} className="input" />
      </div>
      <button className="btn-primary" onClick={save}>
        {saved ? '✓ Saved' : 'Save Changes'}
      </button>
    </div>
  )
}

function BackupRestore() {
  return (
    <div className="max-w-md space-y-4">
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-2">Export Database</h3>
        <p className="text-xs text-slate-500 mb-4">
          Download a consistent snapshot of the SQLite database (device inventory,
          poll history, alerts, and audit log) via SQLite <span className="font-mono">VACUUM INTO</span> —
          safe to run while the server is live.
        </p>
        <a className="btn-secondary" href={api.databaseBackupUrl()} download>
          <Download className="w-4 h-4" /> Export Database
        </a>
      </div>
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-2">Restore Database</h3>
        <p className="text-xs text-slate-500">
          To restore, stop the server and replace the database file at
          {' '}<span className="font-mono text-slate-400">data/embrionix.db</span>{' '}
          with an exported snapshot, then start the server. Live in-place restore is
          intentionally not supported to avoid corrupting an open database.
        </p>
      </div>
    </div>
  )
}

function AboutPage() {
  return (
    <div className="max-w-md space-y-4">
      <div className="card p-5 space-y-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-brand-600 rounded-lg flex items-center justify-center text-white font-bold">E</div>
          <div>
            <div className="font-semibold text-slate-100">Embrionix Dashboard</div>
            <div className="text-xs text-slate-500">Version 0.3.0 — Phase 3</div>
          </div>
        </div>
        <div className="border-t border-surface-700 pt-3 space-y-2 text-xs text-slate-400">
          <div className="flex justify-between">
            <span className="text-slate-500">Backend</span>
            <span className="font-mono">Go · Gin · GORM · SQLite</span>
          </div>
          <div className="flex justify-between">
            <span className="text-slate-500">Frontend</span>
            <span className="font-mono">React · TypeScript · Vite</span>
          </div>
          <div className="flex justify-between">
            <span className="text-slate-500">Device API</span>
            <span className="font-mono">emSFP REST (v1)</span>
          </div>
        </div>
      </div>
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-3">Roadmap</h3>
        {[
          ['Phase 1', 'Foundation — inventory, basic dashboard', 'done'],
          ['Phase 2', 'Monitoring — full EM6 telemetry, reachability, SFP', 'done'],
          ['Phase 3', 'Advanced Monitoring — sparklines, alerts, webhooks, CSV', 'in_progress'],
          ['Phase 4', 'Configuration Management — backup/restore', 'pending'],
          ['Phase 5', 'Enterprise — RBAC, audit logs, notifications', 'pending'],
        ].map(([phase, desc, status]) => (
          <div key={phase} className="flex items-start gap-3 py-2 border-b border-surface-800 last:border-0">
            <span className={clsx(
              'text-xs px-1.5 py-0.5 rounded font-mono shrink-0 mt-0.5',
              status === 'in_progress' ? 'bg-brand-600/20 text-brand-400'
                : status === 'done' ? 'bg-emerald-500/15 text-emerald-400'
                : 'bg-surface-800 text-slate-500',
            )}>
              {status === 'done' ? '✓' : status === 'in_progress' ? '●' : '○'}
            </span>
            <div>
              <div className="text-xs font-medium text-slate-300">{phase}</div>
              <div className="text-xs text-slate-500">{desc}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

const VALID_TABS: Tab[] = ['devices', 'polling', 'alerting', 'bulk', 'backup', 'about']

export function SettingsPage() {
  const { tab } = useParams<{ tab?: string }>()
  const initialTab = (tab && VALID_TABS.includes(tab as Tab) ? tab : 'devices') as Tab
  const [activeTab, setActiveTab] = useState<Tab>(initialTab)
  const navigate = useNavigate()

  const handleTabChange = (t: Tab) => {
    setActiveTab(t)
    navigate(`/settings${t === 'devices' ? '' : `/${t}`}`, { replace: true })
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold text-slate-100">Settings</h1>
        <p className="text-sm text-slate-500 mt-0.5">Application and device configuration</p>
      </div>

      <div className="flex gap-6">
        {/* Sidebar nav */}
        <nav className="w-52 shrink-0 space-y-0.5">
          {TABS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => handleTabChange(id)}
              className={clsx(
                'w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                activeTab === id
                  ? 'bg-surface-700 text-slate-100'
                  : 'text-slate-400 hover:bg-surface-800 hover:text-slate-200',
              )}
            >
              <div className="flex items-center gap-2.5">
                <Icon className="w-4 h-4" />
                {label}
              </div>
              <ChevronRight className="w-3.5 h-3.5 opacity-40" />
            </button>
          ))}
        </nav>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {activeTab === 'devices'  && <DevicesPage />}
          {activeTab === 'polling'  && <PollingSettings />}
          {activeTab === 'alerting' && <AlertingSettings />}
          {activeTab === 'bulk'     && <BulkConfigSettings />}
          {activeTab === 'backup'   && <BackupRestore />}
          {activeTab === 'about'    && <AboutPage />}
        </div>
      </div>
    </div>
  )
}
