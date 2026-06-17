// dBmValue converts a raw SFP optical power reading (µW) to a numeric dBm value,
// or null when there's no signal/module. Formula: dBm = 10 * log10(µW / 1000).
export function dBmValue(rawValue: number | null | undefined): number | null {
  if (!rawValue || rawValue <= 0) return null
  return 10 * Math.log10(rawValue / 1000)
}

// powerTodBm formats a raw µW reading as a "-4.52 dBm" string (or "—").
export function powerTodBm(rawValue: number | null | undefined): string {
  const dbm = dBmValue(rawValue)
  return dbm === null ? '—' : dbm.toFixed(2) + ' dBm'
}

// TX optical power thresholds (dBm) — mirror the server defaults
// (alerting.tx_power_warn_dbm / tx_power_crit_dbm) for on-card colour cues.
export const TX_WARN_DBM = -6
export const TX_CRIT_DBM = -9

// txPowerClass returns a Tailwind text colour for a TX reading by severity.
export function txPowerClass(rawValue: number | null | undefined): string {
  const dbm = dBmValue(rawValue)
  if (dbm === null) return 'text-blue-400'
  if (dbm < TX_CRIT_DBM) return 'text-red-400'
  if (dbm < TX_WARN_DBM) return 'text-amber-400'
  return 'text-blue-400'
}
