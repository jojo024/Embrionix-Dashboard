import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

const NAV_CHORDS: Record<string, string> = {
  d: '/',
  v: '/devices',
  m: '/monitoring',
  s: '/settings',
}

const HELP = [
  ['g then d', 'Go to Dashboard'],
  ['g then v', 'Go to Devices'],
  ['g then m', 'Go to Monitoring'],
  ['g then s', 'Go to Settings'],
  ['n', 'Add device (on Devices page)'],
  ['?', 'Toggle this help'],
  ['Esc', 'Close dialogs'],
]

function isTyping(el: EventTarget | null): boolean {
  const t = el as HTMLElement | null
  return !!t && ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName)
}

// KeyboardShortcuts registers global power-user shortcuts: "g" then a letter to
// navigate, and "?" to toggle a help overlay. Mounted once in the Layout.
export function KeyboardShortcuts() {
  const navigate = useNavigate()
  const [helpOpen, setHelpOpen] = useState(false)
  const [gPending, setGPending] = useState(false)

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (isTyping(e.target) || e.metaKey || e.ctrlKey || e.altKey) return

      if (e.key === '?') { e.preventDefault(); setHelpOpen(o => !o); return }
      if (e.key === 'Escape') { setHelpOpen(false); setGPending(false); return }

      if (gPending) {
        const dest = NAV_CHORDS[e.key]
        if (dest) { e.preventDefault(); navigate(dest) }
        setGPending(false)
        return
      }
      if (e.key === 'g') { setGPending(true); window.setTimeout(() => setGPending(false), 1200) }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [navigate, gPending])

  if (!helpOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={() => setHelpOpen(false)}>
      <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
        <h3 className="text-base font-semibold text-slate-100 mb-4">Keyboard Shortcuts</h3>
        <div className="space-y-2">
          {HELP.map(([keys, desc]) => (
            <div key={keys} className="flex items-center justify-between text-xs">
              <span className="text-slate-400">{desc}</span>
              <kbd className="font-mono px-2 py-0.5 rounded bg-surface-800 border border-surface-700 text-slate-300">{keys}</kbd>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
