import { describe, it, expect, vi, afterEach } from 'vitest'
import { formatRelativeTime } from './time'

describe('formatRelativeTime', () => {
  afterEach(() => vi.useRealTimers())

  function at(now: string) {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(now))
  }

  it('formats seconds', () => {
    at('2026-06-16T12:00:30Z')
    expect(formatRelativeTime('2026-06-16T12:00:00Z')).toBe('30s ago')
  })

  it('formats minutes', () => {
    at('2026-06-16T12:05:00Z')
    expect(formatRelativeTime('2026-06-16T12:00:00Z')).toBe('5m ago')
  })

  it('formats hours', () => {
    at('2026-06-16T15:00:00Z')
    expect(formatRelativeTime('2026-06-16T12:00:00Z')).toBe('3h ago')
  })

  it('falls back to a date for older timestamps', () => {
    at('2026-06-20T12:00:00Z')
    // Older than a day -> locale date string, not a relative "ago" label.
    expect(formatRelativeTime('2026-06-16T12:00:00Z')).not.toContain('ago')
  })
})
