import { Suspense, lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/Layout'
import { Dashboard } from './pages/Dashboard'
import { Login } from './pages/Login'
import { useAuth } from './contexts/AuthContext'

// Lazy-load heavier routes (DeviceDetail and MonitoringPage pull in recharts)
// so they are split out of the initial bundle.
const DeviceDetail = lazy(() => import('./pages/DeviceDetail').then(m => ({ default: m.DeviceDetail })))
const DevicesPage = lazy(() => import('./pages/DevicesPage').then(m => ({ default: m.DevicesPage })))
const MonitoringPage = lazy(() => import('./pages/MonitoringPage').then(m => ({ default: m.MonitoringPage })))
const SettingsPage = lazy(() => import('./pages/SettingsPage').then(m => ({ default: m.SettingsPage })))

function RouteFallback() {
  return <div className="p-8 text-sm text-slate-500">Loading…</div>
}

export default function App() {
  const { loading, authenticated } = useAuth()

  if (loading) {
    return <div className="min-h-screen flex items-center justify-center bg-surface-950 text-sm text-slate-500">Loading…</div>
  }
  if (!authenticated) {
    return <Login />
  }

  return (
    <Layout>
      <Suspense fallback={<RouteFallback />}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/devices" element={<DevicesPage />} />
          <Route path="/devices/:id" element={<DeviceDetail />} />
          <Route path="/monitoring" element={<MonitoringPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/settings/:tab" element={<SettingsPage />} />
        </Routes>
      </Suspense>
    </Layout>
  )
}
