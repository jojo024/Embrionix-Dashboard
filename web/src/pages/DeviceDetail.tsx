import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  ArrowLeft, RefreshCw, Thermometer, Wind, Cpu, Wifi, WifiOff,
  Clock, AlertTriangle, Activity, Server, Radio, Settings2, Download, Sliders
} from 'lucide-react'
import { clsx } from 'clsx'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend
} from 'recharts'
import { useDevice, useDeviceHistory, usePollDevice, useAlertHistory, useAuditLog } from '../hooks/useDevices'
import { downloadWithAuth } from '../api/client'
import { StatusBadge } from '../components/StatusBadge'
import { DeviceConfigTab } from '../components/DeviceConfigTab'
import { useToast } from '../components/Toast'
import { useAuth } from '../contexts/AuthContext'
import { formatDate, formatRelativeTime } from '../utils/time'
import { readableFirmware } from '../utils/firmware'

type Tab = 'overview' | 'interfaces' | 'sfp' | 'monitoring' | 'config' | 'logs'

const TABS: { id: Tab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'overview', label: 'Overview', icon: Server },
  { id: 'interfaces', label: 'Interfaces', icon: Radio },
  { id: 'sfp', label: 'SFP Modules', icon: Activity },
  { id: 'monitoring', label: 'Monitoring', icon: Cpu },
  { id: 'config', label: 'Configuration', icon: Sliders },
  { id: 'logs', label: 'Logs', icon: Settings2 },
]

function InfoRow({ label, value, mono }: { label: string; value?: string | number | null; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between py-2.5 border-b border-surface-800 last:border-0 gap-4">
      <span className="text-xs text-slate-500 shrink-0 w-40">{label}</span>
      <span className={clsx('text-xs text-right break-all', mono ? 'font-mono text-slate-300' : 'text-slate-300')}>
        {value ?? '—'}
      </span>
    </div>
  )
}

function MetricCard({ label, value, unit, warn, icon: Icon }: {
  label: string; value?: number | null; unit: string; warn?: boolean
  icon: React.ComponentType<{ className?: string }>
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-slate-500">{label}</span>
        <Icon className={clsx('w-4 h-4', warn ? 'text-amber-400' : 'text-slate-600')} />
      </div>
      <p className={clsx('text-2xl font-bold font-mono', warn ? 'text-amber-400' : 'text-slate-100')}>
        {value != null ? value.toFixed(value % 1 === 0 ? 0 : 1) : '—'}
        <span className="text-sm font-normal text-slate-500 ml-1">{unit}</span>
      </p>
    </div>
  )
}

function OverviewTab({ device }: { device: ReturnType<typeof useDevice>['data'] }) {
  if (!device) return null
  const pd = device.polling_data

  return (
    <div className="space-y-6">
      {/* Alarms */}
      {pd?.alarms && pd.alarms.length > 0 && (
        <div className="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4 space-y-1">
          <div className="flex items-center gap-2 text-amber-400 font-medium text-sm mb-2">
            <AlertTriangle className="w-4 h-4" /> Active Alarms
          </div>
          {pd.alarms.map((a, i) => (
            <p key={i} className="text-xs text-amber-300/80 font-mono">{a}</p>
          ))}
        </div>
      )}

      {/* Health metrics */}
      {pd && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <MetricCard label="Core Temp" value={pd.core_temp} unit="°C" warn={pd.core_temp > 70} icon={Thermometer} />
          <MetricCard label="Fan Speed" value={pd.fan_speed} unit="RPM" icon={Wind} />
          <MetricCard label="Core Voltage" value={pd.core_voltage ? pd.core_voltage / 1000 : null} unit="V" icon={Cpu} />
          {pd.uptime && (
            <div className="card p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-slate-500">Uptime</span>
                <Clock className="w-4 h-4 text-slate-600" />
              </div>
              <p className="text-lg font-bold text-slate-100 break-words">
                {pd.uptime.split(',')[0]}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Two-column detail */}
      <div className="grid sm:grid-cols-2 gap-4">
        {/* Device info */}
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Device Info</h3>
          <InfoRow label="Name" value={device.name} />
          <InfoRow label="Model" value={device.model || pd?.device_type} />
          <InfoRow label="Serial Number" value={device.serial_number} mono />
          <InfoRow label="Firmware" value={readableFirmware(pd, device.firmware_version)} mono />
          <InfoRow label="emSFP Version" value={pd?.emsfp_version} mono />
          <InfoRow label="Platform HW" value={pd?.platform_hw_version} mono />
          <InfoRow label="Hostname" value={pd?.hostname} mono />
          <InfoRow label="MAC Address" value={pd?.local_mac} mono />
          <InfoRow label="Uptime" value={pd?.uptime} />
        </div>

        {/* Network info */}
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Network</h3>
          <div className="flex items-center justify-between py-2.5 border-b border-surface-800 gap-4">
            <span className="text-xs text-slate-500 shrink-0 w-40">IP (Red)</span>
            <span className="flex items-center gap-2 text-xs font-mono text-slate-300">
              {device.reachable_red != null && (
                <span className={clsx('status-dot', device.reachable_red ? 'status-online' : 'status-offline')} />
              )}
              {device.management_ip_red || '—'}
            </span>
          </div>
          <div className="flex items-center justify-between py-2.5 border-b border-surface-800 gap-4">
            <span className="text-xs text-slate-500 shrink-0 w-40">IP (Blue)</span>
            <span className="flex items-center gap-2 text-xs font-mono text-slate-300">
              {device.reachable_blue != null && (
                <span className={clsx('status-dot', device.reachable_blue ? 'status-online' : 'status-offline')} />
              )}
              {device.management_ip_blue || '—'}
            </span>
          </div>
          <InfoRow label="Active IP" value={pd?.ip_addr} mono />
          <InfoRow label="DHCP" value={pd?.dhcp_enable === '1' ? 'Enabled' : 'Disabled'} />
          <InfoRow label="Location" value={device.location} />
          <InfoRow label="Rack" value={device.rack} />
          <InfoRow label="Last polled" value={device.last_polled_at ? formatDate(device.last_polled_at) : undefined} />
        </div>
      </div>

      {/* PTP + System health */}
      <div className="grid sm:grid-cols-2 gap-4">
        {/* PTP / refclk */}
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">PTP / Reference Clock</h3>
          <div className="flex items-center justify-between py-2.5 border-b border-surface-800 gap-4">
            <span className="text-xs text-slate-500 shrink-0 w-40">Lock Status</span>
            <span className={clsx(
              'text-xs px-2 py-0.5 rounded-full',
              pd?.ptp?.locked ? 'bg-emerald-500/15 text-emerald-400'
                : 'bg-amber-500/15 text-amber-400',
            )}>
              {pd?.ptp?.status_label ?? pd?.refclk_status ?? 'unknown'}
            </span>
          </div>
          <InfoRow label="Master IP" value={pd?.ptp?.master_ip || pd?.grandmaster_id} mono />
          <InfoRow label="Offset from Master" value={pd?.ptp ? `${pd.ptp.offset_from_master} ns` : undefined} mono />
          <InfoRow label="Mean Delay" value={pd?.ptp ? `${pd.ptp.mean_delay} ns` : undefined} mono />
          <InfoRow label="Sync Counter" value={pd?.ptp?.sync_counter} mono />
        </div>

        {/* System health */}
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">System Health</h3>
          <InfoRow label="Video Bandwidth" value={pd?.video_bandwidth_usage} />
          <InfoRow label="Watchdog" value={pd?.watchdog_status} />
          <InfoRow label="IPv4 Packet Drop" value={pd?.ipv4_packet_drop} mono />
          <InfoRow label="Eth RX Errors" value={pd?.ethernet?.rx_error} mono />
          <InfoRow label="SDI Bit Rate" value={pd?.sdi_bit_rate} />
          <InfoRow label="Licensed Features" value={pd?.licenses ? Object.keys(pd.licenses).length : undefined} />
        </div>
      </div>

      {/* Firmware slots */}
      {pd?.firmware_slots && pd.firmware_slots.length > 0 && (
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Firmware Banks</h3>
          <div className="space-y-2">
            {pd.firmware_slots.map(fw => (
              <div key={fw.slot} className="flex items-center gap-3 text-xs">
                <span className="font-mono text-slate-500 w-12">Slot {fw.slot}</span>
                <span className="flex-1 text-slate-300 truncate">{fw.desc || '—'}</span>
                <span className="font-mono text-slate-400">{fw.version || '—'}</span>
                {fw.active && <span className="px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-400">active</span>}
                {fw.default && <span className="px-1.5 py-0.5 rounded bg-brand-500/15 text-brand-400">default</span>}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function InterfacesTab({ device }: { device: ReturnType<typeof useDevice>['data'] }) {
  const pd = device?.polling_data
  if (!pd) {
    return <div className="text-slate-500 text-sm p-4">No interface data available. Device may be offline.</div>
  }
  return (
    <div className="space-y-4">
      {/* Device network interfaces (e1, e2) */}
      {pd.interfaces && pd.interfaces.length > 0 && (
        <div className="grid sm:grid-cols-2 gap-4">
          {pd.interfaces.map(iface => (
            <div key={iface.name} className="card p-4">
              <div className="flex items-center justify-between mb-3">
                <span className="font-medium text-slate-100 font-mono text-sm uppercase">{iface.name}</span>
                <span className={clsx(
                  'text-xs px-2 py-0.5 rounded-full',
                  iface.dhcp ? 'bg-brand-500/15 text-brand-400' : 'bg-slate-700/50 text-slate-400',
                )}>
                  {iface.dhcp ? 'DHCP' : 'Static'}
                </span>
              </div>
              <InfoRow label="Current IP" value={iface.current_ip} mono />
              <InfoRow label="Current Gateway" value={iface.current_gateway} mono />
              <InfoRow label="Static IP" value={iface.static_ip} mono />
              <InfoRow label="VLAN" value={iface.vlan || 'none'} />
            </div>
          ))}
        </div>
      )}

      {/* LLDP neighbour + ethernet counters */}
      <div className="grid sm:grid-cols-2 gap-4">
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">LLDP Neighbours</h3>
          {(() => {
            const neighbors = pd.lldp_neighbors ?? (pd.lldp ? [pd.lldp] : [])
            if (neighbors.length === 0) {
              return <p className="text-xs text-slate-500">No LLDP neighbour discovered.</p>
            }
            return (
              <div className="space-y-3">
                {neighbors.map((n, i) => (
                  <div key={i} className="pb-2 border-b border-surface-800 last:border-0 last:pb-0">
                    <div className="text-xs text-slate-400 mb-1">
                      {n.interface ? `Local port ${n.interface}` : 'Neighbour'}
                    </div>
                    <InfoRow label="Chassis ID" value={n.chassis_id} mono />
                    <InfoRow label="Remote Port" value={n.port_id} mono />
                    <InfoRow label="TTL" value={n.ttl ? `${n.ttl}s` : undefined} />
                  </div>
                ))}
              </div>
            )
          })()}
        </div>
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Control-Plane Ethernet</h3>
          {pd.ethernet ? (
            <>
              <InfoRow label="TX Packets" value={pd.ethernet.tx_packets} mono />
              <InfoRow label="RX Packets" value={pd.ethernet.rx_packets} mono />
              <InfoRow label="RX Errors" value={pd.ethernet.rx_error} mono />
            </>
          ) : (
            <p className="text-xs text-slate-500">No ethernet stats available.</p>
          )}
        </div>
      </div>

      {/* Media flows (telemetry/devices) */}
      {pd.media_devices && pd.media_devices.length > 0 && (
        <div className="card overflow-hidden">
          <div className="px-4 py-3 border-b border-surface-700">
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Media Flows</h3>
          </div>
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-surface-700 text-slate-500">
                <th className="px-4 py-2 text-left font-medium">Device</th>
                <th className="px-4 py-2 text-left font-medium">Type</th>
                <th className="px-4 py-2 text-left font-medium">Channel</th>
                <th className="px-4 py-2 text-left font-medium">Flows</th>
                <th className="px-4 py-2 text-left font-medium">Packets</th>
                <th className="px-4 py-2 text-left font-medium">Valid</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800">
              {pd.media_devices.map(md => (
                <tr key={md.device} className="hover:bg-surface-800/50">
                  <td className="px-4 py-2 font-mono text-slate-400 truncate max-w-[12rem]">{md.device}</td>
                  <td className="px-4 py-2 text-slate-300">{md.type}</td>
                  <td className="px-4 py-2 font-mono text-slate-400">{md.channel}</td>
                  <td className="px-4 py-2 font-mono text-slate-400">{md.flow_count}</td>
                  <td className="px-4 py-2 font-mono text-slate-400">{md.total_pkts.toLocaleString()}</td>
                  <td className="px-4 py-2">
                    <span className={md.valid ? 'text-emerald-400' : 'text-slate-500'}>{md.valid ? '✓' : '✗'}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* SFP port link state */}
      {pd.port_details?.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">SFP Ports</h3>
          {pd.port_details.map(port => (
            <div key={port.port_id} className="card p-4">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  {port.link === 'up'
                    ? <Wifi className="w-4 h-4 text-emerald-400" />
                    : <WifiOff className="w-4 h-4 text-slate-500" />}
                  <span className="font-medium text-slate-100 font-mono text-sm">{port.port_id}</span>
                </div>
                <span className={clsx(
                  'text-xs px-2 py-0.5 rounded-full',
                  port.link === 'up' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-slate-700/50 text-slate-500'
                )}>
                  {port.link ?? 'unknown'}
                </span>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <InfoRow label="Speed" value={port.speed} />
                <InfoRow label="SFP Type" value={port.sfp_type} />
                <InfoRow label="Link" value={port.link} />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function SFPTab({ device }: { device: ReturnType<typeof useDevice>['data'] }) {
  const pd = device?.polling_data
  if (!pd?.port_details?.length) {
    return <div className="text-slate-500 text-sm p-4">No SFP data available.</div>
  }

  return (
    <div className="space-y-4">
      {pd.port_details.map(port => (
        <div key={port.port_id} className="card p-4">
          <h3 className="font-medium text-slate-100 font-mono text-sm mb-4">{port.port_id}</h3>
          {port.ddm ? (
            <div className="grid sm:grid-cols-2 gap-4">
              {/* DDM values */}
              {[
                { label: 'Temperature', v: port.ddm.temperature, unit: '°C' },
                { label: 'VCC', v: port.ddm.vcc, unit: 'V' },
                { label: 'TX Bias', v: port.ddm.tx_bias, unit: 'mA' },
                { label: 'TX Power', v: port.ddm.tx_power, unit: 'µW' },
                { label: 'RX Power', v: port.ddm.rx_power, unit: 'µW' },
              ].map(({ label, v, unit }) => (
                <div key={label} className="bg-surface-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs text-slate-500">{label}</span>
                    <span className="text-sm font-bold font-mono text-slate-100">
                      {v.current.toFixed(2)} <span className="text-xs text-slate-500">{unit}</span>
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-x-4 text-xs text-slate-600">
                    <span>Alarm H: {v.high_alarm}</span>
                    <span>Alarm L: {v.low_alarm}</span>
                    <span>Warn H: {v.high_warning}</span>
                    <span>Warn L: {v.low_warning}</span>
                  </div>
                </div>
              ))}

              {/* Alarm summary */}
              <div className="bg-surface-800 rounded-lg p-3">
                <div className="text-xs text-slate-500 mb-2">Active Alarms</div>
                {Object.entries(port.ddm.alarm_status).some(([, v]) => v) ? (
                  Object.entries(port.ddm.alarm_status)
                    .filter(([, v]) => v)
                    .map(([k]) => (
                      <div key={k} className="text-xs text-red-400 font-mono">{k.replace(/_/g, ' ')}</div>
                    ))
                ) : (
                  <div className="text-xs text-emerald-400">No alarms</div>
                )}
              </div>
            </div>
          ) : (
            <p className="text-xs text-slate-500">No DDM data available for this port.</p>
          )}
        </div>
      ))}
    </div>
  )
}

function MonitoringTab({ deviceId }: { deviceId: string }) {
  const { data: history, isLoading } = useDeviceHistory(deviceId)

  if (isLoading) return <div className="text-slate-500 text-sm p-4">Loading history…</div>
  if (!history?.length) return <div className="text-slate-500 text-sm p-4">No historical data yet. Data is collected after each poll.</div>

  const chartData = [...history].reverse().map(r => ({
    time: new Date(r.polled_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    temp: r.core_temp ?? null,
    p0tx: r.port0_tx_power ?? null,
    p0rx: r.port0_rx_power ?? null,
    ptp: r.ptp_offset ?? null,
    ms: r.response_ms,
  }))

  const hasPtp = chartData.some(d => d.ptp != null)

  return (
    <div className="space-y-6">
      {/* Export */}
      <div className="flex justify-end">
        <button
          className="btn-secondary"
          onClick={() => downloadWithAuth(`/api/v1/devices/${deviceId}/history.csv`, `history-${deviceId}.csv`)}
        >
          <Download className="w-4 h-4" />
          Export CSV
        </button>
      </div>

      {/* Temperature chart */}
      <div className="card p-4">
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Core Temperature (°C)</h3>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
            <XAxis dataKey="time" tick={{ fill: '#64748b', fontSize: 10 }} />
            <YAxis tick={{ fill: '#64748b', fontSize: 10 }} />
            <Tooltip
              contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 8 }}
              labelStyle={{ color: '#94a3b8' }}
              itemStyle={{ color: '#f59e0b' }}
            />
            <Line type="monotone" dataKey="temp" stroke="#f59e0b" dot={false} strokeWidth={2} name="Temp °C" />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* SFP power chart */}
      <div className="card p-4">
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Port 0 SFP Power (µW)</h3>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
            <XAxis dataKey="time" tick={{ fill: '#64748b', fontSize: 10 }} />
            <YAxis tick={{ fill: '#64748b', fontSize: 10 }} />
            <Tooltip
              contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 8 }}
              labelStyle={{ color: '#94a3b8' }}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
            <Line type="monotone" dataKey="p0tx" stroke="#3b82f6" dot={false} strokeWidth={2} name="TX Power" />
            <Line type="monotone" dataKey="p0rx" stroke="#22c55e" dot={false} strokeWidth={2} name="RX Power" />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* PTP offset chart */}
      {hasPtp && (
        <div className="card p-4">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">PTP Offset from Master (ns)</h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
              <XAxis dataKey="time" tick={{ fill: '#64748b', fontSize: 10 }} />
              <YAxis tick={{ fill: '#64748b', fontSize: 10 }} />
              <Tooltip
                contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 8 }}
                labelStyle={{ color: '#94a3b8' }}
                itemStyle={{ color: '#2dd4bf' }}
              />
              <Line type="monotone" dataKey="ptp" stroke="#2dd4bf" dot={false} strokeWidth={2} name="Offset ns" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Response time chart */}
      <div className="card p-4">
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">API Response Time (ms)</h3>
        <ResponsiveContainer width="100%" height={160}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
            <XAxis dataKey="time" tick={{ fill: '#64748b', fontSize: 10 }} />
            <YAxis tick={{ fill: '#64748b', fontSize: 10 }} />
            <Tooltip
              contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 8 }}
              labelStyle={{ color: '#94a3b8' }}
              itemStyle={{ color: '#a78bfa' }}
            />
            <Line type="monotone" dataKey="ms" stroke="#a78bfa" dot={false} strokeWidth={2} name="ms" />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Raw history table */}
      <div className="card overflow-hidden">
        <div className="px-4 py-3 border-b border-surface-700">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Recent Polls</h3>
        </div>
        <div className="overflow-x-auto max-h-64">
          <table className="w-full text-xs">
            <thead className="sticky top-0 bg-surface-900">
              <tr className="border-b border-surface-700">
                {['Time', 'Reachable', 'Response', 'Temp', 'Fan', 'P0 TX', 'P0 RX'].map(h => (
                  <th key={h} className="px-3 py-2 text-left font-medium text-slate-500">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800">
              {history.slice(0, 50).map(r => (
                <tr key={r.id} className="hover:bg-surface-800/50">
                  <td className="px-3 py-2 font-mono text-slate-400">{new Date(r.polled_at).toLocaleTimeString()}</td>
                  <td className="px-3 py-2">
                    <span className={r.reachable ? 'text-emerald-400' : 'text-red-400'}>
                      {r.reachable ? '✓' : '✗'}
                    </span>
                  </td>
                  <td className="px-3 py-2 font-mono text-slate-400">{r.response_ms}ms</td>
                  <td className="px-3 py-2 font-mono text-slate-400">{r.core_temp?.toFixed(1) ?? '—'}</td>
                  <td className="px-3 py-2 font-mono text-slate-400">{r.fan_speed ?? '—'}</td>
                  <td className="px-3 py-2 font-mono text-blue-400">{r.port0_tx_power ?? '—'}</td>
                  <td className="px-3 py-2 font-mono text-green-400">{r.port0_rx_power ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

const STATUS_TEXT: Record<string, string> = {
  online: 'text-emerald-400',
  warning: 'text-amber-400',
  critical: 'text-red-400',
  offline: 'text-slate-400',
  unknown: 'text-slate-500',
}

function LogsTab({ device }: { device: ReturnType<typeof useDevice>['data'] }) {
  const pd = device?.polling_data
  const { data: alertData } = useAlertHistory(device?.id)
  const { data: auditData } = useAuditLog(device?.id)
  const alerts = alertData?.alerts ?? []
  const audit = auditData?.events ?? []

  return (
    <div className="space-y-4">
      {/* Active alarms */}
      <div className="card p-4 space-y-2">
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Active Alarms</h3>
        {pd?.alarms?.length ? (
          pd.alarms.map((a, i) => (
            <div key={i} className="flex items-start gap-2 bg-amber-500/10 border border-amber-500/20 rounded-lg px-3 py-2">
              <AlertTriangle className="w-3.5 h-3.5 text-amber-400 mt-0.5 shrink-0" />
              <span className="text-xs font-mono text-amber-300">{a}</span>
            </div>
          ))
        ) : (
          <p className="text-xs text-slate-500">No active alarms.</p>
        )}
      </div>

      {/* Status-transition history */}
      <div className="card overflow-hidden">
        <div className="px-4 py-3 border-b border-surface-700">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Status History</h3>
        </div>
        {alerts.length === 0 ? (
          <p className="text-xs text-slate-500 p-4">No status changes recorded yet.</p>
        ) : (
          <div className="divide-y divide-surface-800 max-h-96 overflow-y-auto">
            {alerts.map(ev => (
              <div key={ev.id} className="flex items-center gap-3 px-4 py-2.5 text-xs">
                <span className="text-slate-500 font-mono shrink-0 w-32">{formatDate(ev.created_at)}</span>
                <span className="flex items-center gap-1.5">
                  <span className={STATUS_TEXT[ev.from_status]}>{ev.from_status}</span>
                  <span className="text-slate-600">→</span>
                  <span className={STATUS_TEXT[ev.to_status]}>{ev.to_status}</span>
                </span>
                <span className="text-slate-400 truncate">{ev.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Configuration audit log */}
      <div className="card overflow-hidden">
        <div className="px-4 py-3 border-b border-surface-700">
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Configuration Audit Log</h3>
        </div>
        {audit.length === 0 ? (
          <p className="text-xs text-slate-500 p-4">No configuration changes recorded for this device.</p>
        ) : (
          <div className="divide-y divide-surface-800 max-h-96 overflow-y-auto">
            {audit.map(ev => (
              <div key={ev.id} className="flex items-center gap-3 px-4 py-2.5 text-xs">
                <span className="text-slate-500 font-mono shrink-0 w-32">{formatDate(ev.created_at)}</span>
                <span className={clsx('shrink-0', ev.success ? 'text-emerald-400' : 'text-red-400')}>
                  {ev.success ? '✓' : '✗'}
                </span>
                <span className="font-mono text-slate-300 shrink-0">{ev.action}</span>
                <span className="text-slate-500 truncate">{ev.success ? ev.detail : ev.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export function DeviceDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const { data: device, isLoading, refetch, isFetching } = useDevice(id!)
  const pollNow = usePollDevice()
  const { notify } = useToast()
  const { canWrite } = useAuth()

  const handlePollNow = () => {
    pollNow.mutate(id!, {
      onSuccess: (res) => notify(
        res.reachable ? 'success' : 'error',
        res.reachable ? 'Device polled successfully.' : 'Device is unreachable.',
      ),
      onError: (e) => notify('error', `Poll failed: ${(e as Error).message}`),
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-4 animate-pulse">
        <div className="h-8 bg-surface-800 rounded w-48" />
        <div className="h-32 bg-surface-900 border border-surface-700 rounded-xl" />
      </div>
    )
  }

  if (!device) {
    return (
      <div className="card p-12 text-center">
        <p className="text-slate-500 mb-4">Device not found.</p>
        <Link to="/devices" className="btn-secondary">Back to Devices</Link>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Breadcrumb + header (sticky when scrolling) */}
      <div className="sticky top-0 z-30 bg-surface-900/95 backdrop-blur supports-[backdrop-filter]:bg-surface-900/80 -mx-5 px-5 py-4 mb-1 border-b border-surface-800 flex items-start justify-between gap-4">
        <div>
          <button
            className="flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-300 mb-3 transition-colors"
            onClick={() => navigate(-1)}
          >
            <ArrowLeft className="w-3.5 h-3.5" /> Back
          </button>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold text-slate-100">{device.name}</h1>
            <StatusBadge status={device.status} size="md" />
          </div>
          <p className="text-sm text-slate-500 mt-0.5 font-mono">
            {device.management_ip_red}
            {device.last_polled_at && (
              <span className="ml-3 not-mono font-sans">· last polled {formatRelativeTime(device.last_polled_at)}</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {canWrite && (
            <button
              className="btn-secondary"
              onClick={handlePollNow}
              disabled={pollNow.isPending}
            >
              <RefreshCw className={clsx('w-4 h-4', pollNow.isPending && 'animate-spin')} />
              Poll Now
            </button>
          )}
          <button
            className="btn-ghost"
            onClick={() => refetch()}
            disabled={isFetching}
          >
            <RefreshCw className={clsx('w-4 h-4', isFetching && 'animate-spin')} />
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-0.5 border-b border-surface-700 overflow-x-auto">
        {TABS.map(({ id: tid, label, icon: Icon }) => (
          <button
            key={tid}
            onClick={() => setActiveTab(tid)}
            className={clsx(
              'flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium whitespace-nowrap transition-colors border-b-2 -mb-px',
              activeTab === tid
                ? 'text-brand-400 border-brand-500'
                : 'text-slate-500 border-transparent hover:text-slate-300 hover:border-slate-600',
            )}
          >
            <Icon className="w-3.5 h-3.5" />
            {label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div>
        {activeTab === 'overview'    && <OverviewTab device={device} />}
        {activeTab === 'interfaces'  && <InterfacesTab device={device} />}
        {activeTab === 'sfp'         && <SFPTab device={device} />}
        {activeTab === 'monitoring'  && <MonitoringTab deviceId={id!} />}
        {activeTab === 'config'      && <DeviceConfigTab deviceId={id!} active={activeTab === 'config'} />}
        {activeTab === 'logs'        && <LogsTab device={device} />}
      </div>
    </div>
  )
}
