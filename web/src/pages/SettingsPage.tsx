import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Server, Clock, Bell, Download, Info, ChevronRight, Layers, Users, Trash2, Plus, RefreshCw, CheckCircle2, ExternalLink } from 'lucide-react'
import { clsx } from 'clsx'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { DevicesPage } from './DevicesPage'
import { useDevices } from '../hooks/useDevices'
import { useVersion, useCheckUpdate } from '../hooks/useUpdate'
import { useToast } from '../components/Toast'
import { useAuth } from '../contexts/AuthContext'
import { api, downloadWithAuth } from '../api/client'
import { formatRelativeTime } from '../utils/time'
import type { Role } from '../types/device'

type Tab = 'devices' | 'polling' | 'alerting' | 'bulk' | 'backup' | 'users' | 'about'

const TABS: { id: Tab; label: string; icon: React.ComponentType<{ className?: string }>; adminOnly?: boolean }[] = [
  { id: 'devices', label: 'Device Management', icon: Server },
  { id: 'polling', label: 'Polling Configuration', icon: Clock },
  { id: 'alerting', label: 'Alerting', icon: Bell },
  { id: 'bulk', label: 'Bulk Configuration', icon: Layers },
  { id: 'backup', label: 'Backup & Restore', icon: Download },
  { id: 'users', label: 'Users & Access', icon: Users, adminOnly: true },
  { id: 'about', label: 'About', icon: Info },
]

function UsersSettings() {
  const { authEnabled } = useAuth()
  const { notify } = useToast()
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['users'], queryFn: () => api.listUsers(), enabled: authEnabled })
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('viewer')

  const reload = () => qc.invalidateQueries({ queryKey: ['users'] })

  if (!authEnabled) {
    return (
      <div className="max-w-md card p-4 text-sm text-slate-400">
        Authentication is currently <strong>disabled</strong>, so the dashboard runs with no
        login and full access. Enable it in <span className="font-mono">configs/config.yaml</span>{' '}
        (<span className="font-mono">auth.enabled: true</span> + a <span className="font-mono">jwt_secret</span>)
        to manage users and roles here.
      </div>
    )
  }

  const create = async () => {
    try {
      await api.createUser(username, password, role)
      notify('success', `User "${username}" created.`)
      setUsername(''); setPassword(''); setRole('viewer'); reload()
    } catch (e) { notify('error', `Create failed: ${(e as Error).message}`) }
  }
  const changeRole = async (id: number, r: Role) => {
    try { await api.updateUser(id, { role: r }); notify('success', 'Role updated.'); reload() }
    catch (e) { notify('error', `Update failed: ${(e as Error).message}`) }
  }
  const remove = async (id: number, name: string) => {
    try { await api.deleteUser(id); notify('success', `Deleted "${name}".`); reload() }
    catch (e) { notify('error', `Delete failed: ${(e as Error).message}`) }
  }

  return (
    <div className="max-w-2xl space-y-5">
      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-surface-700 text-xs text-slate-500">
              <th className="px-4 py-2 text-left font-medium">Username</th>
              <th className="px-4 py-2 text-left font-medium">Role</th>
              <th className="px-4 py-2 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-800">
            {(data?.users ?? []).map(u => (
              <tr key={u.id} className="hover:bg-surface-800/50">
                <td className="px-4 py-2 text-slate-200">{u.username}</td>
                <td className="px-4 py-2">
                  <select className="input py-1 text-xs w-32" value={u.role} onChange={e => changeRole(u.id, e.target.value as Role)}>
                    <option value="viewer">viewer</option>
                    <option value="operator">operator</option>
                    <option value="admin">admin</option>
                  </select>
                </td>
                <td className="px-4 py-2 text-right">
                  <button className="btn-ghost p-1.5 hover:text-red-400" onClick={() => remove(u.id, u.username)} title="Delete">
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card p-4 space-y-3">
        <h3 className="text-sm font-medium text-slate-100">Add User</h3>
        <div className="grid grid-cols-3 gap-3">
          <input className="input" placeholder="Username" value={username} onChange={e => setUsername(e.target.value)} />
          <input className="input" type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} />
          <select className="input" value={role} onChange={e => setRole(e.target.value as Role)}>
            <option value="viewer">viewer</option>
            <option value="operator">operator</option>
            <option value="admin">admin</option>
          </select>
        </div>
        <button className="btn-primary" onClick={create} disabled={!username || !password}>
          <Plus className="w-4 h-4" /> Add User
        </button>
      </div>
    </div>
  )
}

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
  const { notify } = useToast()
  const dl = (path: string, name: string) =>
    downloadWithAuth(path, name).catch(e => notify('error', `Download failed: ${(e as Error).message}`))

  return (
    <div className="max-w-md space-y-4">
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-2">Export Database</h3>
        <p className="text-xs text-slate-500 mb-4">
          Download a consistent snapshot of the SQLite database (device inventory,
          poll history, alerts, and audit log) via SQLite <span className="font-mono">VACUUM INTO</span> —
          safe to run while the server is live.
        </p>
        <button className="btn-secondary" onClick={() => dl('/api/v1/backup', 'embrionix.db')}>
          <Download className="w-4 h-4" /> Export Database
        </button>
      </div>
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-2">Export Ansible Inventory</h3>
        <p className="text-xs text-slate-500 mb-4">
          Download the device inventory as Ansible dynamic-inventory JSON
          (group <span className="font-mono">emsfp</span>, with per-host model, location, and IPs).
        </p>
        <button className="btn-secondary" onClick={() => dl('/api/v1/export/ansible', 'embrionix-inventory.json')}>
          <Download className="w-4 h-4" /> Export Inventory
        </button>
      </div>
      <div className="card p-4">
        <h3 className="text-sm font-medium text-slate-100 mb-2">Fleet Report</h3>
        <p className="text-xs text-slate-500 mb-4">
          Download a PDF summary of fleet status, active alarms, and recent status
          changes. A scheduled text summary can also be delivered to the alerting
          webhook (see <span className="font-mono">reports</span> in config).
        </p>
        <button className="btn-secondary" onClick={() => dl('/api/v1/report.pdf', 'embrionix-fleet-report.pdf')}>
          <Download className="w-4 h-4" /> Download Report (PDF)
        </button>
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
  const { data: ver } = useVersion()
  const { canWrite } = useAuth()
  const { notify } = useToast()
  const checkUpdate = useCheckUpdate()

  const handleCheck = () => {
    checkUpdate.mutate(undefined, {
      onSuccess: (status) => {
        if (status.error) {
          notify('error', `Update check failed: ${status.error}`)
        } else if (status.update_available) {
          notify('success', `Update ${status.latest_version} is available.`)
        } else {
          notify('info', "You're running the latest version.")
        }
      },
      onError: (e) => notify('error', `Update check failed: ${(e as Error).message}`),
    })
  }

  return (
    <div className="max-w-md space-y-4">
      <div className="card p-5 space-y-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-brand-600 rounded-lg flex items-center justify-center text-white font-bold">E</div>
          <div>
            <div className="font-semibold text-slate-100">Embrionix Dashboard</div>
            <div className="text-xs text-slate-500">
              Version {ver?.current_version ?? '—'}
              {ver?.update_available && (
                <span className="ml-2 text-brand-400">· update {ver.latest_version} available</span>
              )}
            </div>
          </div>
        </div>

        {/* Check for updates */}
        <div className="border-t border-surface-700 pt-3 space-y-2">
          {canWrite ? (
            <button
              onClick={handleCheck}
              disabled={checkUpdate.isPending}
              className="btn-secondary text-xs py-1.5"
            >
              <RefreshCw className={clsx('w-3.5 h-3.5', checkUpdate.isPending && 'animate-spin')} />
              {checkUpdate.isPending ? 'Checking…' : 'Check for updates'}
            </button>
          ) : (
            <p className="text-[11px] text-slate-500">An operator or admin can check for updates.</p>
          )}

          {ver && !checkUpdate.isPending && (
            ver.update_available ? (
              <div className="flex items-center gap-2 text-xs text-brand-400">
                <Download className="w-3.5 h-3.5 shrink-0" />
                <span>
                  Update {ver.latest_version} available
                  {ver.release_url && (
                    <a
                      href={ver.release_url}
                      target="_blank"
                      rel="noreferrer"
                      className="ml-2 inline-flex items-center gap-0.5 hover:text-brand-300"
                    >
                      release notes <ExternalLink className="w-3 h-3" />
                    </a>
                  )}
                </span>
              </div>
            ) : checkUpdate.isSuccess || ver.checked_at ? (
              <div className="flex items-center gap-2 text-xs text-emerald-400">
                <CheckCircle2 className="w-3.5 h-3.5 shrink-0" />
                <span>Up to date</span>
              </div>
            ) : null
          )}

          {ver?.checked_at && (
            <p className="text-[11px] text-slate-600">Last checked {formatRelativeTime(ver.checked_at)}</p>
          )}
          {ver && !ver.enabled && (
            <p className="text-[11px] text-amber-500/80">Automatic update checks are disabled in configuration.</p>
          )}
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
    </div>
  )
}

const VALID_TABS: Tab[] = ['devices', 'polling', 'alerting', 'bulk', 'backup', 'users', 'about']

export function SettingsPage() {
  const { tab } = useParams<{ tab?: string }>()
  const { isAdmin } = useAuth()
  const initialTab = (tab && VALID_TABS.includes(tab as Tab) ? tab : 'devices') as Tab
  const [activeTab, setActiveTab] = useState<Tab>(initialTab)
  const navigate = useNavigate()
  const visibleTabs = TABS.filter(t => !t.adminOnly || isAdmin)

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

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Sidebar nav — horizontal scroll on mobile, sidebar on desktop */}
        <nav className="w-full lg:w-52 shrink-0 flex lg:block gap-1 lg:gap-0 lg:space-y-0.5 overflow-x-auto">
          {visibleTabs.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => handleTabChange(id)}
              className={clsx(
                'shrink-0 lg:w-full flex items-center justify-between gap-2 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors whitespace-nowrap',
                activeTab === id
                  ? 'bg-surface-700 text-slate-100'
                  : 'text-slate-400 hover:bg-surface-800 hover:text-slate-200',
              )}
            >
              <div className="flex items-center gap-2.5">
                <Icon className="w-4 h-4 shrink-0" />
                {label}
              </div>
              <ChevronRight className="w-3.5 h-3.5 opacity-40 hidden lg:block" />
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
          {activeTab === 'users'    && <UsersSettings />}
          {activeTab === 'about'    && <AboutPage />}
        </div>
      </div>
    </div>
  )
}
