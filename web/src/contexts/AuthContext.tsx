import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { api, setAuthToken, clearAuthToken, getAuthToken } from '../api/client'
import type { Role } from '../types/device'

interface AuthState {
  loading: boolean
  authEnabled: boolean
  authenticated: boolean
  username: string | null
  role: Role | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  /** operator or admin */
  canWrite: boolean
  isAdmin: boolean
}

const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [loading, setLoading] = useState(true)
  const [authEnabled, setAuthEnabled] = useState(false)
  const [username, setUsername] = useState<string | null>(null)
  const [role, setRole] = useState<Role | null>(null)

  const refresh = useCallback(async () => {
    try {
      const me = await api.getMe()
      setAuthEnabled(me.auth_enabled)
      setUsername(me.username || null)
      setRole(me.role || null)
    } catch {
      // 401 → unauthenticated; auth is enabled but we have no valid token.
      setAuthEnabled(true)
      setUsername(null)
      setRole(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const onUnauthorized = () => { setUsername(null); setRole(null); setAuthEnabled(true) }
    window.addEventListener('emb:unauthorized', onUnauthorized)
    return () => window.removeEventListener('emb:unauthorized', onUnauthorized)
  }, [refresh])

  const login = useCallback(async (u: string, p: string) => {
    const res = await api.login(u, p)
    setAuthToken(res.token)
    setUsername(res.user.username)
    setRole(res.user.role)
    setAuthEnabled(true)
  }, [])

  const logout = useCallback(() => {
    clearAuthToken()
    setUsername(null)
    setRole(null)
  }, [])

  const value: AuthState = {
    loading,
    authEnabled,
    // Authenticated when auth is disabled (implicit admin) or we have a session.
    authenticated: !authEnabled || (!!getAuthToken() && !!role),
    username,
    role,
    login,
    logout,
    canWrite: !authEnabled || role === 'operator' || role === 'admin',
    isAdmin: !authEnabled || role === 'admin',
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
