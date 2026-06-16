import { useState } from 'react'
import { Radio, LogIn } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'

export function Login() {
  const { login } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(username, password)
    } catch (err) {
      setError((err as Error).message || 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface-950 p-4">
      <form onSubmit={submit} className="card p-8 w-full max-w-sm space-y-5">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-brand-600 rounded-lg flex items-center justify-center">
            <Radio className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="font-semibold text-slate-100">Embrionix Dashboard</div>
            <div className="text-xs text-slate-500">Sign in to continue</div>
          </div>
        </div>

        {error && (
          <div className="text-xs text-red-400 bg-red-500/10 border border-red-500/30 rounded-lg px-3 py-2">{error}</div>
        )}

        <div>
          <label className="label">Username</label>
          <input className="input" value={username} autoFocus autoComplete="username"
            onChange={e => setUsername(e.target.value)} />
        </div>
        <div>
          <label className="label">Password</label>
          <input className="input" type="password" value={password} autoComplete="current-password"
            onChange={e => setPassword(e.target.value)} />
        </div>

        <button className="btn-primary w-full justify-center" disabled={busy || !username || !password}>
          <LogIn className="w-4 h-4" />
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
