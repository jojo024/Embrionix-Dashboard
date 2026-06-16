import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { CheckCircle2, AlertTriangle, Info, X } from 'lucide-react'
import { clsx } from 'clsx'

export type ToastKind = 'success' | 'error' | 'info'

interface Toast {
  id: number
  kind: ToastKind
  message: string
}

interface ToastContextValue {
  notify: (kind: ToastKind, message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

// useToast returns a notify(kind, message) helper. Safe to call from anywhere
// under <ToastProvider>.
export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within a ToastProvider')
  return ctx
}

let nextId = 1

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const dismiss = useCallback((id: number) => {
    setToasts(t => t.filter(x => x.id !== id))
  }, [])

  const notify = useCallback((kind: ToastKind, message: string) => {
    const id = nextId++
    setToasts(t => [...t, { id, kind, message }])
  }, [])

  return (
    <ToastContext.Provider value={{ notify }}>
      {children}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-80 max-w-[calc(100vw-2rem)]">
        {toasts.map(t => (
          <ToastItem key={t.id} toast={t} onDismiss={() => dismiss(t.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

const ICONS = {
  success: CheckCircle2,
  error: AlertTriangle,
  info: Info,
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const Icon = ICONS[toast.kind]

  useEffect(() => {
    const timer = window.setTimeout(onDismiss, 4000)
    return () => window.clearTimeout(timer)
  }, [onDismiss])

  return (
    <div
      role="status"
      className={clsx(
        'flex items-start gap-3 rounded-lg border px-3.5 py-3 shadow-lg backdrop-blur',
        'animate-[slideIn_0.15s_ease-out]',
        toast.kind === 'success' && 'bg-emerald-500/10 border-emerald-500/30',
        toast.kind === 'error' && 'bg-red-500/10 border-red-500/30',
        toast.kind === 'info' && 'bg-surface-800 border-surface-700',
      )}
    >
      <Icon className={clsx(
        'w-4 h-4 mt-0.5 shrink-0',
        toast.kind === 'success' && 'text-emerald-400',
        toast.kind === 'error' && 'text-red-400',
        toast.kind === 'info' && 'text-brand-400',
      )} />
      <p className="flex-1 text-xs text-slate-200 leading-relaxed">{toast.message}</p>
      <button onClick={onDismiss} className="text-slate-500 hover:text-slate-300 shrink-0">
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  )
}
