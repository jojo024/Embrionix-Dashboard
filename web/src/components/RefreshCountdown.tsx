import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'

interface Props {
  /** Refetch interval in seconds. */
  intervalSeconds: number
  /** react-query dataUpdatedAt timestamp (ms). */
  lastUpdated: number
  /** True while a refetch is in flight. */
  isFetching: boolean
}

// RefreshCountdown shows how many seconds remain until the next auto-refresh,
// driven by react-query's dataUpdatedAt so it stays in sync with actual fetches.
export function RefreshCountdown({ intervalSeconds, lastUpdated, isFetching }: Props) {
  const [remaining, setRemaining] = useState(intervalSeconds)

  useEffect(() => {
    const tick = () => {
      const elapsed = (Date.now() - lastUpdated) / 1000
      setRemaining(Math.max(0, Math.ceil(intervalSeconds - elapsed)))
    }
    tick()
    const timer = window.setInterval(tick, 1000)
    return () => window.clearInterval(timer)
  }, [intervalSeconds, lastUpdated])

  return (
    <span className="flex items-center gap-1.5 text-xs text-slate-500 tabular-nums">
      <RefreshCw className={clsx('w-3 h-3', isFetching && 'animate-spin text-brand-400')} />
      {isFetching ? 'refreshing…' : `next in ${remaining}s`}
    </span>
  )
}
