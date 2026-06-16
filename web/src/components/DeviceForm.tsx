import { useState, useEffect } from 'react'
import { X, Loader } from 'lucide-react'
import type { Device } from '../types/device'

type FormData = Omit<Device, 'id' | 'created_at' | 'updated_at' | 'status' | 'last_polled_at' | 'reachable_red' | 'reachable_blue' | 'polling_data' | 'slow_response_count'>

const EMPTY: FormData = {
  name: '',
  description: '',
  location: '',
  rack: '',
  serial_number: '',
  model: '',
  firmware_version: '',
  management_ip_red: '',
  management_ip_blue: '',
  tags: '',
  notes: '',
  monitoring_enabled: true,
}

interface Props {
  device?: Device | null
  onSubmit: (data: FormData) => void
  onCancel: () => void
  isLoading?: boolean
}

function Field({ label, name, value, onChange, placeholder, required, type = 'text' }: {
  label: string
  name: string
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void
  placeholder?: string
  required?: boolean
  type?: string
}) {
  return (
    <div>
      <label className="label">{label}{required && <span className="text-red-400 ml-0.5">*</span>}</label>
      <input
        type={type}
        name={name}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        required={required}
        className="input"
      />
    </div>
  )
}

export function DeviceForm({ device, onSubmit, onCancel, isLoading }: Props) {
  const [form, setForm] = useState<FormData>(EMPTY)
  const [fetchingFirmware, setFetchingFirmware] = useState(false)

  useEffect(() => {
    if (device) {
      setForm({
        name: device.name,
        description: device.description,
        location: device.location,
        rack: device.rack,
        serial_number: device.serial_number,
        model: device.model,
        firmware_version: device.firmware_version,
        management_ip_red: device.management_ip_red,
        management_ip_blue: device.management_ip_blue,
        tags: device.tags,
        notes: device.notes,
        monitoring_enabled: device.monitoring_enabled,
      })
    } else {
      setForm(EMPTY)
    }
  }, [device])

  // Auto-fetch firmware version when creating a new device and IP is set
  useEffect(() => {
    if (device || !form.management_ip_red && !form.management_ip_blue) return

    const ip = form.management_ip_red || form.management_ip_blue
    if (!ip) return

    const timer = setTimeout(async () => {
      setFetchingFirmware(true)
      try {
        const response = await fetch(`http://${ip}/emsfp/node/v1/self/information`, {
          signal: AbortSignal.timeout(5000)
        })
        if (response.ok) {
          const data = await response.json()
          setForm(prev => ({
            ...prev,
            firmware_version: data.current_version || prev.firmware_version
          }))
        }
      } catch {
        // Silently fail if device is unreachable
      } finally {
        setFetchingFirmware(false)
      }
    }, 800)

    return () => clearTimeout(timer)
  }, [form.management_ip_red, form.management_ip_blue, device])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value, type } = e.target
    setForm(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? (e.target as HTMLInputElement).checked : value,
    }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(form)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div className="bg-surface-900 border border-surface-700 rounded-xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-surface-700">
          <h2 className="text-base font-semibold text-slate-100">
            {device ? 'Edit Device' : 'Add Device'}
          </h2>
          <button className="btn-ghost p-1" onClick={onCancel}>
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto">
          <div className="px-6 py-5 space-y-5">
            {/* Identity */}
            <section>
              <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Identity</h3>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Device Name" name="name" value={form.name} onChange={handleChange} placeholder="EM6-MCR-01" required />
                <Field label="Model" name="model" value={form.model} onChange={handleChange} placeholder="Embox6" />
                <div className="col-span-2">
                  <Field label="Description" name="description" value={form.description} onChange={handleChange} placeholder="Short description" />
                </div>
              </div>
            </section>

            {/* Location */}
            <section>
              <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Location</h3>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Location" name="location" value={form.location} onChange={handleChange} placeholder="MCR, Rack Room A" />
                <Field label="Rack" name="rack" value={form.rack} onChange={handleChange} placeholder="Rack 3, Unit 12" />
              </div>
            </section>

            {/* Networking */}
            <section>
              <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Management Network</h3>
              <div className="grid grid-cols-2 gap-3">
                <Field label="IP Address (Red)" name="management_ip_red" value={form.management_ip_red} onChange={handleChange} placeholder="192.168.1.100" required={!device} />
                <Field label="IP Address (Blue)" name="management_ip_blue" value={form.management_ip_blue} onChange={handleChange} placeholder="192.168.2.100" required={!device} />
              </div>
            </section>

            {/* Hardware */}
            <section>
              <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Hardware</h3>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Serial Number" name="serial_number" value={form.serial_number} onChange={handleChange} placeholder="EMB-2024-XXXXX" />
                <div>
                  <label className="label">
                    Firmware Version
                    {!device && <span className="text-slate-500 text-xs ml-1">(auto-detected)</span>}
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      name="firmware_version"
                      value={form.firmware_version}
                      onChange={handleChange}
                      placeholder="Auto-fetched from device"
                      readOnly={!device}
                      className="input"
                    />
                    {fetchingFirmware && (
                      <Loader className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 animate-spin" />
                    )}
                  </div>
                </div>
              </div>
            </section>

            {/* Tags & Notes */}
            <section>
              <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Additional Info</h3>
              <div className="space-y-3">
                <Field label="Tags (comma-separated)" name="tags" value={form.tags} onChange={handleChange} placeholder="production, 2110, encoding" />
                <div>
                  <label className="label">Notes</label>
                  <textarea
                    name="notes"
                    value={form.notes}
                    onChange={handleChange}
                    rows={3}
                    placeholder="Additional notes about this device..."
                    className="input resize-none"
                  />
                </div>
              </div>
            </section>

            {/* Monitoring */}
            <section>
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  name="monitoring_enabled"
                  checked={form.monitoring_enabled}
                  onChange={handleChange}
                  className="w-4 h-4 rounded accent-brand-500"
                />
                <div>
                  <div className="text-sm text-slate-200 font-medium">Enable Monitoring</div>
                  <div className="text-xs text-slate-500">Poll this device on the configured interval</div>
                </div>
              </label>
            </section>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-surface-700">
            <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={isLoading}>
              {isLoading ? 'Saving…' : device ? 'Save Changes' : 'Add Device'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
