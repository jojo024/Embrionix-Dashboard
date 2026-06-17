import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Server, Settings, Activity, Menu, X, Radio, LogOut, UserCircle } from 'lucide-react';
import { useState } from 'react';
import { clsx } from 'clsx';
import { useApiStatus } from '../hooks/useApiStatus';
import { useAuth } from '../contexts/AuthContext';
import { KeyboardShortcuts } from './KeyboardShortcuts';
import { UpdateNotification } from './UpdateNotification';
import { useVersion } from '../hooks/useUpdate';

const NAV = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/devices', label: 'Devices', icon: Server },
  { to: '/monitoring', label: 'Monitoring', icon: Activity },
  { to: '/settings', label: 'Settings', icon: Settings },
];

interface Props { children: React.ReactNode }

export function Layout({ children }: Props) {
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const apiConnected = useApiStatus();
  const { authEnabled, username, role, logout } = useAuth();
  const { data: version } = useVersion();

  return (
    <div className="flex h-screen bg-surface-950 overflow-hidden">
      {/* Sidebar */}
      <aside
        className={clsx(
          'fixed inset-y-0 left-0 z-40 flex flex-col w-60 bg-surface-900 border-r border-surface-700',
          'transition-transform duration-200 lg:translate-x-0 lg:static lg:inset-auto',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        {/* Logo */}
        <div className="flex items-center gap-3 px-5 h-16 border-b border-surface-700">
          <div className="w-8 h-8 bg-brand-600 rounded-lg flex items-center justify-center">
            <Radio className="w-4 h-4 text-white" />
          </div>
          <div>
            <div className="text-sm font-semibold text-slate-100">Embrionix</div>
            <div className="text-xs text-slate-500">Dashboard</div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-0.5">
          {NAV.map(({ to, label, icon: Icon }) => {
            const active = to === '/' ? location.pathname === '/' : location.pathname.startsWith(to);
            return (
              <Link
                key={to}
                to={to}
                onClick={() => setSidebarOpen(false)}
                className={clsx(
                  'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  active
                    ? 'bg-brand-600/20 text-brand-400 border border-brand-600/30'
                    : 'text-slate-400 hover:bg-surface-800 hover:text-slate-200',
                )}
              >
                <Icon className="w-4 h-4 shrink-0" />
                {label}
              </Link>
            );
          })}
        </nav>

        {/* Footer */}
        <div className="px-5 py-4 border-t border-surface-700">
          <p className="text-xs text-slate-600">Embrionix Dashboard{version?.current_version ? ` · ${version.current_version}` : ''}</p>
        </div>
      </aside>

      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top bar */}
        <header className="flex items-center gap-3 h-16 px-6 bg-surface-900 border-b border-surface-700 shrink-0">
          <button
            className="lg:hidden btn-ghost p-1.5"
            onClick={() => setSidebarOpen(!sidebarOpen)}
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
          <div className="flex-1" />
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <span className={clsx('status-dot', apiConnected ? 'status-online' : 'status-offline')} />
            <span className="hidden sm:inline">{apiConnected ? 'API connected' : 'API disconnected'}</span>
          </div>
          {authEnabled && username && (
            <div className="flex items-center gap-2 ml-3 pl-3 border-l border-surface-700">
              <UserCircle className="w-4 h-4 text-slate-500 shrink-0" />
              <span className="text-xs text-slate-300 hidden sm:inline">{username}</span>
              <span className="text-xs px-1.5 py-0.5 rounded-full bg-surface-800 text-slate-400 capitalize hidden sm:inline">{role}</span>
              <button className="btn-ghost p-1.5" onClick={logout} title="Sign out">
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          )}
        </header>

        <main className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>

      <KeyboardShortcuts />
      <UpdateNotification />
    </div>
  );
}
