import { useState } from 'react';
import { Download, X, RefreshCw, ExternalLink, AlertCircle, CheckCircle2 } from 'lucide-react';
import { api } from '../api/client';
import { useVersion, useApplyUpdate } from '../hooks/useUpdate';
import { useAuth } from '../contexts/AuthContext';

const DISMISS_KEY = 'emb:dismissed-update';

type Phase = 'prompt' | 'confirm' | 'updating' | 'error';

// UpdateNotification shows a pop-up when a newer release is available, with
// Update (admin) and Dismiss actions. The Update button triggers a server
// self-update + restart, then waits for the server to come back and reloads.
export function UpdateNotification() {
  const { data } = useVersion();
  const { isAdmin } = useAuth();
  const applyUpdate = useApplyUpdate();
  const [phase, setPhase] = useState<Phase>('prompt');
  const [error, setError] = useState('');
  const [dismissedVersion, setDismissedVersion] = useState<string>(
    () => localStorage.getItem(DISMISS_KEY) ?? '',
  );

  if (!data?.update_available) return null;
  if (dismissedVersion === data.latest_version) return null;

  const dismiss = () => {
    localStorage.setItem(DISMISS_KEY, data.latest_version);
    setDismissedVersion(data.latest_version);
  };

  const runUpdate = async () => {
    setPhase('updating');
    setError('');
    try {
      await applyUpdate.mutateAsync();
      // The server is restarting. Poll /health until it returns the new version,
      // then reload. Fall back to a reload after ~60s regardless.
      await waitForRestart(data.latest_version);
      window.location.reload();
    } catch (e) {
      setError((e as Error).message);
      setPhase('error');
    }
  };

  return (
    <div className="fixed bottom-4 right-4 z-50 w-80 bg-surface-900 border border-brand-500/40 rounded-xl shadow-2xl shadow-black/40 overflow-hidden">
      <div className="px-4 py-3 bg-brand-500/10 border-b border-brand-500/20 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Download className="w-4 h-4 text-brand-400" />
          <span className="text-sm font-semibold text-slate-100">Update available</span>
        </div>
        {phase !== 'updating' && (
          <button onClick={dismiss} className="btn-ghost p-1" title="Dismiss">
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      <div className="px-4 py-3 space-y-3">
        <div className="text-xs text-slate-400">
          <span className="font-mono text-slate-500">{data.current_version}</span>
          {' → '}
          <span className="font-mono text-emerald-400">{data.latest_version}</span>
          {data.release_url && (
            <a
              href={data.release_url}
              target="_blank"
              rel="noreferrer"
              className="ml-2 inline-flex items-center gap-0.5 text-brand-400 hover:text-brand-300"
            >
              release notes <ExternalLink className="w-3 h-3" />
            </a>
          )}
        </div>

        {phase === 'updating' && (
          <div className="flex items-center gap-2 text-xs text-amber-300">
            <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            Updating &amp; restarting — the page will reload automatically…
          </div>
        )}

        {phase === 'error' && (
          <div className="flex items-start gap-2 text-xs text-red-400">
            <AlertCircle className="w-3.5 h-3.5 mt-0.5 shrink-0" />
            <span>{error || 'Update failed.'}</span>
          </div>
        )}

        {phase === 'confirm' && (
          <div className="text-xs text-amber-300/90 bg-amber-500/10 rounded-md px-2 py-1.5">
            This restarts the server (~10s of monitoring downtime). Continue?
          </div>
        )}

        {phase !== 'updating' && (
          <div className="flex items-center justify-end gap-2">
            <button onClick={dismiss} className="btn-secondary text-xs py-1">Dismiss</button>
            {isAdmin && phase !== 'confirm' && phase !== 'error' && (
              <button onClick={() => setPhase('confirm')} className="btn-primary text-xs py-1">
                <Download className="w-3.5 h-3.5" /> Update
              </button>
            )}
            {isAdmin && phase === 'confirm' && (
              <button onClick={runUpdate} className="btn-primary text-xs py-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> Confirm
              </button>
            )}
            {isAdmin && phase === 'error' && (
              <button onClick={runUpdate} className="btn-primary text-xs py-1">Retry</button>
            )}
          </div>
        )}

        {!isAdmin && (
          <p className="text-[11px] text-slate-500">An administrator must apply the update.</p>
        )}
      </div>
    </div>
  );
}

// waitForRestart polls /health until it reports the target version (server back
// up on the new build), or gives up after ~60s.
async function waitForRestart(targetVersion: string): Promise<void> {
  const deadline = Date.now() + 60_000;
  // Brief initial delay so we don't catch the old process before it exits.
  await sleep(2000);
  while (Date.now() < deadline) {
    try {
      const h = await api.health();
      if (h.version === targetVersion) return;
    } catch {
      // server is mid-restart; keep waiting
    }
    await sleep(2000);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
