import { useState } from 'react'
import { RefreshCw, Lock, Pencil, X, Power, RotateCcw } from 'lucide-react'
import { clsx } from 'clsx'
import {
  useDeviceConfig, useUpdateNetwork, useUpdateProtocols, useUpdateSyslog,
  useUpdateRoutes, useRebootDevice, useConfigReset,
} from '../hooks/useDevices'
import { useToast } from './Toast'
import { ConfirmDialog } from './ConfirmDialog'
import type {
  NetworkConfig, ProtocolsConfig, SyslogConfig, StaticRoute, ConfigResetScope,
} from '../types/device'

function Row({ label, value, mono }: { label: string; value?: string | number | null; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between py-2.5 border-b border-surface-800 last:border-0 gap-4">
      <span className="text-xs text-slate-500 shrink-0 w-40">{label}</span>
      <span className={clsx('text-xs text-right break-all', mono ? 'font-mono text-slate-300' : 'text-slate-300')}>
        {value ?? '—'}
      </span>
    </div>
  )
}

function Field({ label, value, onChange, placeholder, type = 'text' }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string
}) {
  return (
    <div>
      <label className="label">{label}</label>
      <input className="input" type={type} value={value} placeholder={placeholder}
        onChange={e => onChange(e.target.value)} />
    </div>
  )
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 cursor-pointer py-1">
      <span className="text-xs text-slate-400">{label}</span>
      <button
        type="button"
        onClick={() => onChange(!checked)}
        className={clsx('relative w-9 h-5 rounded-full transition-colors', checked ? 'bg-brand-600' : 'bg-surface-700')}
      >
        <span className={clsx('absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform', checked ? 'translate-x-4' : 'translate-x-0.5')} />
      </button>
    </label>
  )
}

function SectionHeader({ title, editing, onEdit, onCancel }: {
  title: string; editing: boolean; onEdit: () => void; onCancel: () => void
}) {
  return (
    <div className="flex items-center justify-between mb-3">
      <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">{title}</h3>
      {editing ? (
        <button className="btn-ghost p-1" onClick={onCancel} title="Cancel"><X className="w-3.5 h-3.5" /></button>
      ) : (
        <button className="btn-ghost p-1 text-slate-500 hover:text-slate-300" onClick={onEdit} title="Edit">
          <Pencil className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  )
}

export function DeviceConfigTab({ deviceId, active }: { deviceId: string; active: boolean }) {
  const { data: config, isLoading, isError, error, refetch, isFetching } = useDeviceConfig(deviceId, active)
  const { notify } = useToast()

  if (isLoading) return <div className="text-slate-500 text-sm p-4">Loading configuration…</div>
  if (isError) {
    return (
      <div className="card p-4">
        <p className="text-sm text-red-400 mb-3">Failed to read configuration: {(error as Error).message}</p>
        <button className="btn-secondary" onClick={() => refetch()}><RefreshCw className="w-4 h-4" /> Retry</button>
      </div>
    )
  }
  if (!config) return <div className="text-slate-500 text-sm p-4">No configuration available.</div>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 bg-surface-800/60 border border-surface-700 rounded-lg px-4 py-2.5">
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Lock className="w-3.5 h-3.5 text-slate-500" />
          Changes are written to the live device and recorded in the audit log. Network changes reboot the device.
        </div>
        <button className="btn-ghost p-1.5" onClick={() => refetch()} disabled={isFetching} title="Refresh">
          <RefreshCw className={clsx('w-4 h-4', isFetching && 'animate-spin')} />
        </button>
      </div>

      <div className="grid sm:grid-cols-2 gap-4">
        {config.network && <NetworkSection deviceId={deviceId} data={config.network} notify={notify} />}
        {config.protocols && <ProtocolsSection deviceId={deviceId} data={config.protocols} notify={notify} />}
        {config.syslog && <SyslogSection deviceId={deviceId} data={config.syslog} notify={notify} />}
        {config.dns && (
          <div className="card p-4">
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">DNS (read-only)</h3>
            <Row label="Server" value={config.dns.server_address} mono />
            <Row label="Domain" value={config.dns.domain_name || '—'} mono />
          </div>
        )}
        <RoutesSection deviceId={deviceId} data={config.static_routes ?? []} notify={notify} />
      </div>

      <DeviceActions deviceId={deviceId} notify={notify} />
    </div>
  )
}

type Notify = (kind: 'success' | 'error' | 'info', msg: string) => void

function NetworkSection({ deviceId, data, notify }: { deviceId: string; data: NetworkConfig; notify: Notify }) {
  const [editing, setEditing] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const [form, setForm] = useState(data)
  const mut = useUpdateNetwork(deviceId)
  const dhcp = form.dhcp_enable === '1'

  const save = () => {
    mut.mutate({
      ip_addr: form.ip_addr, subnet_mask: form.subnet_mask, gateway: form.gateway,
      hostname: form.hostname, port: form.port, dhcp_enable: form.dhcp_enable,
      ctl_vlan_id: form.ctl_vlan_id, ctl_vlan_pcp: form.ctl_vlan_pcp, ctl_vlan_enable: form.ctl_vlan_enable,
    }, {
      onSuccess: () => { notify('success', 'Network settings sent. The device is rebooting to apply.'); setEditing(false); setConfirm(false) },
      onError: (e) => { notify('error', `Network update failed: ${(e as Error).message}`); setConfirm(false) },
    })
  }

  return (
    <div className="card p-4">
      <SectionHeader title="Network (ipconfig)" editing={editing} onEdit={() => { setForm(data); setEditing(true) }} onCancel={() => setEditing(false)} />
      {!editing ? (
        <>
          <Row label="MAC Address" value={data.mac_address} mono />
          <Row label="IP Address" value={data.ip_addr} mono />
          <Row label="Subnet Mask" value={data.subnet_mask} mono />
          <Row label="Gateway" value={data.gateway} mono />
          <Row label="Hostname" value={data.hostname} mono />
          <Row label="HTTP Port" value={data.port} mono />
          <Row label="DHCP" value={dhcp ? 'Enabled' : 'Disabled'} />
          <Row label="Control VLAN" value={data.ctl_vlan_enable === '1' ? `${data.ctl_vlan_id} (pcp ${data.ctl_vlan_pcp})` : 'Disabled'} mono />
        </>
      ) : (
        <div className="space-y-3">
          <Toggle label="DHCP" checked={dhcp} onChange={v => setForm({ ...form, dhcp_enable: v ? '1' : '0' })} />
          {!dhcp && (
            <>
              <Field label="IP Address" value={form.ip_addr} onChange={v => setForm({ ...form, ip_addr: v })} placeholder="192.168.1.50" />
              <Field label="Subnet Mask" value={form.subnet_mask} onChange={v => setForm({ ...form, subnet_mask: v })} placeholder="255.255.255.0" />
              <Field label="Gateway" value={form.gateway} onChange={v => setForm({ ...form, gateway: v })} placeholder="192.168.1.1" />
            </>
          )}
          <Field label="Hostname" value={form.hostname} onChange={v => setForm({ ...form, hostname: v })} />
          <Field label="HTTP Port" value={form.port} onChange={v => setForm({ ...form, port: v })} placeholder="80" />
          <Toggle label="Control VLAN" checked={form.ctl_vlan_enable === '1'} onChange={v => setForm({ ...form, ctl_vlan_enable: v ? '1' : '0' })} />
          {form.ctl_vlan_enable === '1' && (
            <div className="grid grid-cols-2 gap-2">
              <Field label="VLAN ID" value={form.ctl_vlan_id} onChange={v => setForm({ ...form, ctl_vlan_id: v })} />
              <Field label="VLAN PCP" value={form.ctl_vlan_pcp} onChange={v => setForm({ ...form, ctl_vlan_pcp: v })} />
            </div>
          )}
          <button className="btn-primary w-full" onClick={() => setConfirm(true)}>Save & Reboot</button>
        </div>
      )}
      {confirm && (
        <ConfirmDialog
          danger
          title="Apply network settings?"
          message={<>The device will <strong>reboot</strong> to apply the new network configuration. If the IP changes, update this device's management IP afterwards or it will appear offline.</>}
          confirmLabel="Apply & Reboot"
          busy={mut.isPending}
          onConfirm={save}
          onCancel={() => setConfirm(false)}
        />
      )}
    </div>
  )
}

function ProtocolsSection({ deviceId, data, notify }: { deviceId: string; data: ProtocolsConfig; notify: Notify }) {
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState(data)
  const mut = useUpdateProtocols(deviceId)

  const save = () => {
    mut.mutate(form, {
      onSuccess: () => { notify('success', 'Protocol settings updated.'); setEditing(false) },
      onError: (e) => notify('error', `Update failed: ${(e as Error).message}`),
    })
  }

  return (
    <div className="card p-4">
      <SectionHeader title="Protocols" editing={editing} onEdit={() => { setForm(data); setEditing(true) }} onCancel={() => setEditing(false)} />
      {!editing ? (
        <>
          <Row label="mDNS" value={data.mdns_enable === '1' ? 'Enabled' : 'Disabled'} />
          <Row label="Ember+ Port" value={data.ember_server_port} mono />
          <Row label="SAP Announce" value={data.sap_announcement_enable === '1' ? 'Enabled' : 'Disabled'} />
        </>
      ) : (
        <div className="space-y-3">
          <Toggle label="mDNS" checked={form.mdns_enable === '1'} onChange={v => setForm({ ...form, mdns_enable: v ? '1' : '0' })} />
          <Field label="Ember+ Port" value={form.ember_server_port} onChange={v => setForm({ ...form, ember_server_port: v })} />
          <Toggle label="SAP Announce" checked={form.sap_announcement_enable === '1'} onChange={v => setForm({ ...form, sap_announcement_enable: v ? '1' : '0' })} />
          <button className="btn-primary w-full" onClick={save} disabled={mut.isPending}>
            {mut.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      )}
    </div>
  )
}

function SyslogSection({ deviceId, data, notify }: { deviceId: string; data: SyslogConfig; notify: Notify }) {
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({ server: data.server, port: String(data.port), enable: data.enable })
  const mut = useUpdateSyslog(deviceId)

  const save = () => {
    mut.mutate(
      { server: form.server, port: Number(form.port) || 514, enable: form.enable, monitoring: data.monitoring },
      {
        onSuccess: () => { notify('success', 'Syslog settings updated.'); setEditing(false) },
        onError: (e) => notify('error', `Update failed: ${(e as Error).message}`),
      },
    )
  }

  return (
    <div className="card p-4">
      <SectionHeader title="Syslog" editing={editing}
        onEdit={() => { setForm({ server: data.server, port: String(data.port), enable: data.enable }); setEditing(true) }}
        onCancel={() => setEditing(false)} />
      {!editing ? (
        <>
          <Row label="Status" value={data.enable ? 'Enabled' : 'Disabled'} />
          <Row label="Server" value={data.server} mono />
          <Row label="Port" value={data.port} mono />
        </>
      ) : (
        <div className="space-y-3">
          <Toggle label="Enable" checked={form.enable} onChange={v => setForm({ ...form, enable: v })} />
          <Field label="Server" value={form.server} onChange={v => setForm({ ...form, server: v })} placeholder="192.168.1.10" />
          <Field label="Port" value={form.port} onChange={v => setForm({ ...form, port: v })} placeholder="514" />
          <button className="btn-primary w-full" onClick={save} disabled={mut.isPending}>
            {mut.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      )}
    </div>
  )
}

function RoutesSection({ deviceId, data, notify }: { deviceId: string; data: StaticRoute[]; notify: Notify }) {
  const [editing, setEditing] = useState(false)
  const [routes, setRoutes] = useState<StaticRoute[]>(data)
  const mut = useUpdateRoutes(deviceId)

  const save = () => {
    mut.mutate(routes.filter(r => r.destination && r.gateway), {
      onSuccess: () => { notify('success', 'Static routes updated.'); setEditing(false) },
      onError: (e) => notify('error', `Update failed: ${(e as Error).message}`),
    })
  }

  return (
    <div className="card p-4">
      <SectionHeader title="Static Routes" editing={editing}
        onEdit={() => { setRoutes(data.length ? data : [{ name: 'route_1', destination: '', gateway: '' }]); setEditing(true) }}
        onCancel={() => setEditing(false)} />
      {!editing ? (
        data.length === 0
          ? <p className="text-xs text-slate-500">No static routes configured.</p>
          : data.map(r => <Row key={r.name} label={r.destination} value={`→ ${r.gateway}`} mono />)
      ) : (
        <div className="space-y-3">
          {routes.map((r, i) => (
            <div key={i} className="flex items-end gap-2">
              <div className="flex-1">
                <Field label={i === 0 ? 'Destination (CIDR)' : ''} value={r.destination}
                  onChange={v => setRoutes(routes.map((x, j) => j === i ? { ...x, destination: v } : x))} placeholder="192.168.5.0/24" />
              </div>
              <div className="flex-1">
                <Field label={i === 0 ? 'Gateway' : ''} value={r.gateway}
                  onChange={v => setRoutes(routes.map((x, j) => j === i ? { ...x, gateway: v } : x))} placeholder="192.168.1.1" />
              </div>
              <button className="btn-ghost p-2 mb-0.5 hover:text-red-400" onClick={() => setRoutes(routes.filter((_, j) => j !== i))}>
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          ))}
          {routes.length < 5 && (
            <button className="btn-secondary w-full" onClick={() => setRoutes([...routes, { name: `route_${routes.length + 1}`, destination: '', gateway: '' }])}>
              Add route
            </button>
          )}
          <button className="btn-primary w-full" onClick={save} disabled={mut.isPending}>
            {mut.isPending ? 'Saving…' : 'Save routes'}
          </button>
        </div>
      )}
    </div>
  )
}

const RESET_SCOPES: { scope: ConfigResetScope; label: string; desc: string }[] = [
  { scope: 'flows', label: 'Flows', desc: 'Reset all flow configuration to factory.' },
  { scope: 'application', label: 'Application', desc: 'Reset application config (keeps interfaces, license).' },
  { scope: 'generic', label: 'Generic', desc: 'Reset generic config (license, secondary interface).' },
  { scope: 'system', label: 'System (full)', desc: 'Complete factory reset of the entire device.' },
]

function DeviceActions({ deviceId, notify }: { deviceId: string; notify: Notify }) {
  const [showReboot, setShowReboot] = useState(false)
  const [resetScope, setResetScope] = useState<ConfigResetScope | null>(null)
  const reboot = useRebootDevice(deviceId)
  const reset = useConfigReset(deviceId)

  const doReboot = () => reboot.mutate(undefined, {
    onSuccess: () => { notify('success', 'Reboot requested.'); setShowReboot(false) },
    onError: (e) => { notify('error', `Reboot failed: ${(e as Error).message}`); setShowReboot(false) },
  })

  const doReset = () => {
    if (!resetScope) return
    reset.mutate(resetScope, {
      onSuccess: () => { notify('success', `Config reset (${resetScope}) requested.`); setResetScope(null) },
      onError: (e) => { notify('error', `Reset failed: ${(e as Error).message}`); setResetScope(null) },
    })
  }

  return (
    <div className="card p-4 border-red-500/20">
      <h3 className="text-xs font-semibold text-red-400/80 uppercase tracking-wider mb-3">Device Actions</h3>
      <div className="flex flex-wrap gap-2">
        <button className="btn-secondary" onClick={() => setShowReboot(true)}>
          <Power className="w-4 h-4" /> Reboot
        </button>
        {RESET_SCOPES.map(s => (
          <button key={s.scope} className="btn-secondary hover:text-red-400" onClick={() => setResetScope(s.scope)}>
            <RotateCcw className="w-4 h-4" /> Reset {s.label}
          </button>
        ))}
      </div>

      {showReboot && (
        <ConfirmDialog danger title="Reboot device?"
          message="The device will restart and be briefly unreachable. In-progress media flows will be interrupted."
          confirmLabel="Reboot" busy={reboot.isPending}
          onConfirm={doReboot} onCancel={() => setShowReboot(false)} />
      )}
      {resetScope && (
        <ConfirmDialog danger title={`Reset configuration (${resetScope})?`}
          message={<>{RESET_SCOPES.find(s => s.scope === resetScope)?.desc} The device will reboot. <strong>This cannot be undone.</strong></>}
          confirmLabel={`Reset ${resetScope}`} busy={reset.isPending}
          onConfirm={doReset} onCancel={() => setResetScope(null)} />
      )}
    </div>
  )
}
