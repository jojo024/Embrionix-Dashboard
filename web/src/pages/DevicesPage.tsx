import { useEffect, useState } from 'react'
import { Plus, Pencil, Trash2, Search, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'
import { useDevices, useCreateDevice, useUpdateDevice, useDeleteDevice } from '../hooks/useDevices'
import { DeviceForm } from '../components/DeviceForm'
import { StatusBadge } from '../components/StatusBadge'
import { useToast } from '../components/Toast'
import type { Device } from '../types/device'
import { formatDate } from '../utils/time'

export function DevicesPage() {
  const { data, isLoading, refetch } = useDevices()
  const createDevice = useCreateDevice()
  const updateDevice = useUpdateDevice()
  const deleteDevice = useDeleteDevice()

  const { notify } = useToast()
  const [showForm, setShowForm] = useState(false)
  const [editTarget, setEditTarget] = useState<Device | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null)
  const [search, setSearch] = useState('')

  // Keyboard shortcut: press "n" to open the Add Device form (ignored while typing).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (e.key === 'n' && !e.metaKey && !e.ctrlKey && !e.altKey &&
        !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) {
        e.preventDefault()
        setEditTarget(null)
        setShowForm(true)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const devices = (data?.devices ?? []).filter(d =>
    !search ||
    d.name.toLowerCase().includes(search.toLowerCase()) ||
    d.management_ip_red.includes(search) ||
    d.location.toLowerCase().includes(search.toLowerCase())
  )

  const handleCreate = async (form: Omit<Device, 'id' | 'created_at' | 'updated_at' | 'status' | 'last_polled_at' | 'reachable_red' | 'reachable_blue' | 'polling_data'>) => {
    try {
      await createDevice.mutateAsync(form)
      setShowForm(false)
      notify('success', `Device "${form.name}" added.`)
    } catch (e) {
      notify('error', `Failed to add device: ${(e as Error).message}`)
    }
  }

  const handleUpdate = async (form: typeof editTarget & object) => {
    if (!editTarget) return
    try {
      await updateDevice.mutateAsync({ ...editTarget, ...form })
      setEditTarget(null)
      setShowForm(false)
      notify('success', `Device "${editTarget.name}" updated.`)
    } catch (e) {
      notify('error', `Failed to update device: ${(e as Error).message}`)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    const name = deleteTarget.name
    try {
      await deleteDevice.mutateAsync(deleteTarget.id)
      setDeleteTarget(null)
      notify('success', `Device "${name}" deleted.`)
    } catch (e) {
      notify('error', `Failed to delete device: ${(e as Error).message}`)
    }
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-100">Device Inventory</h1>
          <p className="text-sm text-slate-500 mt-0.5">{data?.total ?? 0} devices registered</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="btn-secondary" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4" />
          </button>
          <button className="btn-primary" onClick={() => { setEditTarget(null); setShowForm(true) }}>
            <Plus className="w-4 h-4" />
            Add Device
          </button>
        </div>
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
        <input
          type="text"
          placeholder="Search devices…"
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="input pl-9"
        />
      </div>

      {/* Table */}
      <div className="card overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-slate-500">Loading devices…</div>
        ) : devices.length === 0 ? (
          <div className="p-12 text-center">
            <p className="text-slate-500 mb-4">
              {search ? 'No devices match your search.' : 'No devices yet.'}
            </p>
            {!search && (
              <button className="btn-primary" onClick={() => setShowForm(true)}>
                <Plus className="w-4 h-4" /> Add your first device
              </button>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-surface-700">
                  {['Status', 'Name', 'Model', 'Location / Rack', 'IP Red', 'IP Blue', 'Monitoring', 'Added', 'Actions'].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-medium text-slate-500 whitespace-nowrap">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800">
                {devices.map(device => (
                  <tr key={device.id} className="hover:bg-surface-800/50 transition-colors">
                    <td className="px-4 py-3">
                      <StatusBadge status={device.status} />
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-100">{device.name}</div>
                      {device.description && (
                        <div className="text-xs text-slate-500 truncate max-w-[180px]">{device.description}</div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-slate-400 text-xs font-mono">{device.model || '—'}</td>
                    <td className="px-4 py-3 text-slate-400">
                      {[device.location, device.rack].filter(Boolean).join(' / ') || '—'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">{device.management_ip_red || '—'}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">{device.management_ip_blue || '—'}</td>
                    <td className="px-4 py-3">
                      <span className={clsx(
                        'text-xs px-2 py-0.5 rounded-full',
                        device.monitoring_enabled
                          ? 'bg-emerald-500/10 text-emerald-400'
                          : 'bg-slate-700/50 text-slate-500'
                      )}>
                        {device.monitoring_enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-slate-500 whitespace-nowrap">
                      {formatDate(device.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <button
                          className="btn-ghost p-1.5"
                          title="Edit"
                          onClick={() => { setEditTarget(device); setShowForm(true) }}
                        >
                          <Pencil className="w-3.5 h-3.5" />
                        </button>
                        <button
                          className="btn-ghost p-1.5 hover:text-red-400"
                          title="Delete"
                          onClick={() => setDeleteTarget(device)}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Add / Edit form modal */}
      {showForm && (
        <DeviceForm
          device={editTarget}
          onSubmit={editTarget ? handleUpdate as never : handleCreate}
          onCancel={() => { setShowForm(false); setEditTarget(null) }}
          isLoading={createDevice.isPending || updateDevice.isPending}
        />
      )}

      {/* Delete confirm dialog */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
          <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-sm shadow-2xl">
            <h3 className="text-base font-semibold text-slate-100 mb-2">Delete device?</h3>
            <p className="text-sm text-slate-400 mb-5">
              <span className="font-medium text-slate-200">{deleteTarget.name}</span> will be permanently removed
              along with all historical poll data. This cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button className="btn-secondary" onClick={() => setDeleteTarget(null)}>Cancel</button>
              <button className="btn-danger" onClick={handleDelete} disabled={deleteDevice.isPending}>
                {deleteDevice.isPending ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
